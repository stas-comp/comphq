// Package tasks holds the Board, Team view and My jobs (SPEC A6).
package tasks

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// The four Board columns, in display order (SPEC gate 2.01).
const (
	StageIdea  = "idea"
	StageTodo  = "todo"
	StageDoing = "doing"
	StageDone  = "done"
)

// Stages lists every stage in board display order.
var Stages = []string{StageIdea, StageTodo, StageDoing, StageDone}

// doneRetentionDays is how long a finished task still shows on the board
// before quietly leaving it (SPEC gate 2.08).
const doneRetentionDays = 14

// Task is one row of the tasks table, with its assignees resolved.
type Task struct {
	ID        int64
	Title     string
	Notes     string
	Size      string // S | M | L
	Stage     string
	Position  int
	DueDate   string // "YYYY-MM-DD", empty if unset
	Assignees []Assignee
	Overdue   bool
}

// Assignee is one person assigned to a task, as shown on its card. A
// removed person still shows on their old tasks, marked (SPEC gate 2.11).
type Assignee struct {
	PersonID int64
	Name     string
	Removed  bool
}

// ErrEmptyTitle is returned by Create for blank or whitespace-only input.
var ErrEmptyTitle = errors.New("title can't be empty")

// ErrInvalidSize/ErrInvalidStage are returned by Create for a value
// outside the fixed enums (SPEC B3).
var (
	ErrInvalidSize  = errors.New("that isn't a valid size")
	ErrInvalidStage = errors.New("that isn't a valid stage")
)

// Store reads and writes tasks, their assignees and their activity.
type Store struct {
	DB *sql.DB
}

// CreateInput is what adding a task takes (SPEC gate 2.02).
type CreateInput struct {
	Title     string
	Notes     string
	Size      string // "" defaults to M
	Stage     string // "" defaults to idea
	DueDate   string // "" for none
	PersonIDs []int64
}

func validSize(size string) bool {
	return size == "S" || size == "M" || size == "L"
}

func validStage(stage string) bool {
	for _, s := range Stages {
		if s == stage {
			return true
		}
	}
	return false
}

// Create adds a task at the bottom of its stage and records a "created"
// activity row, all in one transaction (SPEC gate 2.02).
func (s *Store) Create(ctx context.Context, input CreateInput, creatorID int64, now time.Time) (Task, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Task{}, ErrEmptyTitle
	}
	size := input.Size
	if size == "" {
		size = "M"
	}
	if !validSize(size) {
		return Task{}, ErrInvalidSize
	}
	stage := input.Stage
	if stage == "" {
		stage = StageIdea
	}
	if !validStage(stage) {
		return Task{}, ErrInvalidStage
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	var maxPos sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(position) FROM tasks WHERE stage = ? AND removed_at IS NULL`, stage,
	).Scan(&maxPos); err != nil {
		return Task{}, err
	}
	position := int(maxPos.Int64) + 1

	nowStr := now.UTC().Format(time.RFC3339)
	var dueDate any
	if input.DueDate != "" {
		dueDate = input.DueDate
	}
	// A task created straight into Done just became done, right now — set
	// done_at immediately rather than leaving it NULL, which ListBoard's
	// 14-day retention query would otherwise treat as "unknown" and
	// exclude outright (SQL NULL comparisons are neither true nor false).
	var doneAt any
	if stage == StageDone {
		doneAt = nowStr
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO tasks (title, notes, size, stage, position, due_date, done_at, created_by, created_at, updated_by, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		title, input.Notes, size, stage, position, dueDate, doneAt, creatorID, nowStr, creatorID, nowStr,
	)
	if err != nil {
		return Task{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Task{}, err
	}

	for _, personID := range input.PersonIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO task_assignees (task_id, person_id) VALUES (?, ?)`, id, personID,
		); err != nil {
			return Task{}, err
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO task_activity (task_id, person_id, action, detail, at) VALUES (?, ?, 'created', '{}', ?)`,
		id, creatorID, nowStr,
	); err != nil {
		return Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return Task{}, err
	}

	return s.Get(ctx, id, now)
}

// Get returns one task by id, with its assignees resolved and Overdue
// computed against today.
func (s *Store) Get(ctx context.Context, id int64, today time.Time) (Task, error) {
	var t Task
	var dueDate, doneAt sql.NullString
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, title, notes, size, stage, position, due_date, done_at FROM tasks WHERE id = ?`, id,
	).Scan(&t.ID, &t.Title, &t.Notes, &t.Size, &t.Stage, &t.Position, &dueDate, &doneAt)
	if err != nil {
		return Task{}, err
	}
	t.DueDate = dueDate.String
	t.Overdue = isOverdue(t.Stage, t.DueDate, today)

	assignees, err := s.assigneesFor(ctx, []int64{id})
	if err != nil {
		return Task{}, err
	}
	t.Assignees = assignees[id]
	return t, nil
}

// ListBoard returns every task that belongs on the Board (SPEC gate
// 2.01): not removed, and not done for more than 14 days (gate 2.08),
// ordered by stage then position.
func (s *Store) ListBoard(ctx context.Context, today time.Time) ([]Task, error) {
	cutoff := today.AddDate(0, 0, -doneRetentionDays).UTC().Format(time.RFC3339)

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, title, notes, size, stage, position, due_date, done_at
		FROM tasks
		WHERE removed_at IS NULL
		  AND NOT (stage = 'done' AND done_at IS NOT NULL AND done_at < ?)
		ORDER BY
			CASE stage WHEN 'idea' THEN 0 WHEN 'todo' THEN 1 WHEN 'doing' THEN 2 WHEN 'done' THEN 3 END,
			position
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	var ids []int64
	for rows.Next() {
		var t Task
		var dueDate, doneAt sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Notes, &t.Size, &t.Stage, &t.Position, &dueDate, &doneAt); err != nil {
			return nil, err
		}
		t.DueDate = dueDate.String
		t.Overdue = isOverdue(t.Stage, t.DueDate, today)
		tasks = append(tasks, t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	assigneesByTask, err := s.assigneesFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].Assignees = assigneesByTask[tasks[i].ID]
	}
	return tasks, nil
}

// isOverdue reports SPEC gate 2.03's "unfinished task past its due
// date": done tasks are never overdue regardless of date.
func isOverdue(stage, dueDate string, today time.Time) bool {
	if dueDate == "" || stage == StageDone {
		return false
	}
	return dueDate < today.Format("2006-01-02")
}

// assigneesFor returns each task's assignees, keyed by task id, in a
// deterministic (person id) order. A removed person still appears,
// marked (SPEC gate 2.11).
func (s *Store) assigneesFor(ctx context.Context, taskIDs []int64) (map[int64][]Assignee, error) {
	result := make(map[int64][]Assignee, len(taskIDs))
	if len(taskIDs) == 0 {
		return result, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(taskIDs)), ",")
	args := make([]any, len(taskIDs))
	for i, id := range taskIDs {
		args[i] = id
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT ta.task_id, p.id, p.name, p.active
		FROM task_assignees ta
		JOIN people p ON p.id = ta.person_id
		WHERE ta.task_id IN (`+placeholders+`)
		ORDER BY ta.task_id, p.id
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var taskID int64
		var a Assignee
		var active int
		if err := rows.Scan(&taskID, &a.PersonID, &a.Name, &active); err != nil {
			return nil, err
		}
		a.Removed = active == 0
		result[taskID] = append(result[taskID], a)
	}
	return result, rows.Err()
}
