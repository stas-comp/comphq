// Package tasks holds the Board, Team view and My jobs (SPEC A6).
package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
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
	// StepsTotal and StepsDone count the job's live steps, for the small
	// 3/7 on a card (gate 5.10). Set by ListBoard only.
	StepsTotal int
	StepsDone  int
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
	// ErrInvalidDate is a due date that isn't a real day (SPEC gate 4.05: the
	// date field is typed, so nothing but the server can check it).
	ErrInvalidDate = errors.New("that isn't a valid date")
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
	if input.DueDate != "" && !format.ValidISODate(input.DueDate) {
		return Task{}, ErrInvalidDate
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
	if err := bumpTasksVersion(ctx, tx); err != nil {
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

// ErrTaskNotFound is returned by Move for a removed or non-existent id.
var ErrTaskNotFound = errors.New("that task isn't there any more")

// ErrInvalidMove is returned by Move when the caller didn't specify
// exactly one of BeforeID, AfterID or ToBottom (SPEC B4).
var ErrInvalidMove = errors.New("exactly one of before_id, after_id or to_bottom is required")

// MoveInput is one move (SPEC B4, gate 2.04's button path): Stage is the
// destination stage (the same as the task's current one, for a same-
// stage reorder), and exactly one of BeforeID/AfterID/ToBottom places it
// within that stage.
type MoveInput struct {
	Stage    string
	BeforeID int64
	AfterID  int64
	ToBottom bool
}

// Move renumbers position for the affected stage(s) in one transaction
// (SPEC B4), recording a "moved" activity only when the stage actually
// changes (PLAN.md P2-02).
func (s *Store) Move(ctx context.Context, taskID int64, input MoveInput, actorID int64, now time.Time) error {
	if !validStage(input.Stage) {
		return ErrInvalidStage
	}
	targets := 0
	if input.BeforeID != 0 {
		targets++
	}
	if input.AfterID != 0 {
		targets++
	}
	if input.ToBottom {
		targets++
	}
	if targets != 1 {
		return ErrInvalidMove
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldStage string
	err = tx.QueryRowContext(ctx, `SELECT stage FROM tasks WHERE id = ? AND removed_at IS NULL`, taskID).Scan(&oldStage)
	if err == sql.ErrNoRows {
		return ErrTaskNotFound
	}
	if err != nil {
		return err
	}

	targetOthers, err := stageOrder(ctx, tx, input.Stage, taskID)
	if err != nil {
		return err
	}
	newTargetOrder, err := insertTask(targetOthers, taskID, input.BeforeID, input.AfterID, input.ToBottom)
	if err != nil {
		return err
	}
	if err := renumber(ctx, tx, newTargetOrder); err != nil {
		return err
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET stage = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		input.Stage, actorID, nowStr, taskID,
	); err != nil {
		return err
	}
	if err := updateDoneAtForStageChange(ctx, tx, taskID, oldStage, input.Stage, nowStr); err != nil {
		return err
	}

	if oldStage != input.Stage {
		oldOthers, err := stageOrder(ctx, tx, oldStage, taskID)
		if err != nil {
			return err
		}
		if err := renumber(ctx, tx, oldOthers); err != nil {
			return err
		}

		detail := fmt.Sprintf(`{"from":%q,"to":%q}`, oldStage, input.Stage)
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO task_activity (task_id, person_id, action, detail, at) VALUES (?, ?, 'moved', ?, ?)`,
			taskID, actorID, detail, nowStr,
		); err != nil {
			return err
		}
	}

	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// updateDoneAtForStageChange keeps done_at in sync with a stage change
// that entered or left Done. ListBoard's 14-day retention (gate 2.08)
// needs a real timestamp the moment a task lands in Done, not just from
// Create (which already sets it for a task created straight into Done)
// — a card dragged or moved into Done later needs the same treatment,
// and one moved back out again should stop being tracked as finished.
func updateDoneAtForStageChange(ctx context.Context, tx *sql.Tx, taskID int64, oldStage, newStage, nowStr string) error {
	switch {
	case oldStage == newStage:
		return nil
	case newStage == StageDone:
		_, err := tx.ExecContext(ctx, `UPDATE tasks SET done_at = ? WHERE id = ?`, nowStr, taskID)
		return err
	case oldStage == StageDone:
		_, err := tx.ExecContext(ctx, `UPDATE tasks SET done_at = NULL WHERE id = ?`, taskID)
		return err
	default:
		return nil
	}
}

// stageOrder returns a stage's current tasks, in position order,
// excluding excludeID (the task being moved — irrelevant when it isn't
// in this stage at all, but harmless either way).
func stageOrder(ctx context.Context, tx *sql.Tx, stage string, excludeID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id FROM tasks WHERE stage = ? AND removed_at IS NULL AND id != ? ORDER BY position`,
		stage, excludeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// renumber writes 1..n positions for orderedIDs, in that order — the
// "renumbers position for the affected stage in one transaction" SPEC B4
// describes. Simple full-stage renumbering rather than shifting only the
// tasks between old and new spots: boards are small enough that this
// costs nothing, and it can never drift into a duplicate or a gap.
func renumber(ctx context.Context, tx *sql.Tx, orderedIDs []int64) error {
	for i, id := range orderedIDs {
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET position = ? WHERE id = ?`, i+1, id); err != nil {
			return err
		}
	}
	return nil
}

// BoardFilter narrows ListBoard (SPEC gate 2.07): PersonID (0 = no
// filter) is resolved by the caller from either "My tasks" or a chosen
// person — the store only ever filters by one concrete id — and Query
// (empty = no filter) matches anywhere in the title, case-insensitive.
type BoardFilter struct {
	PersonID int64
	Query    string
}

// ListBoard returns every task that belongs on the Board (SPEC gate
// 2.01): not removed, and not done for more than 14 days (gate 2.08),
// ordered by stage then position, narrowed by filter (SPEC gate 2.07).
func (s *Store) ListBoard(ctx context.Context, today time.Time, filter BoardFilter) ([]Task, error) {
	cutoff := today.AddDate(0, 0, -doneRetentionDays).UTC().Format(time.RFC3339)

	query := `
		SELECT id, title, notes, size, stage, position, due_date, done_at
		FROM tasks
		WHERE removed_at IS NULL
		  AND NOT (stage = 'done' AND done_at IS NOT NULL AND done_at < ?)
	`
	args := []any{cutoff}
	if filter.PersonID != 0 {
		query += ` AND id IN (SELECT task_id FROM task_assignees WHERE person_id = ?)`
		args = append(args, filter.PersonID)
	}
	if filter.Query != "" {
		query += ` AND LOWER(title) LIKE '%' || LOWER(?) || '%'`
		args = append(args, filter.Query)
	}
	query += `
		ORDER BY
			CASE stage WHEN 'idea' THEN 0 WHEN 'todo' THEN 1 WHEN 'doing' THEN 2 WHEN 'done' THEN 3 END,
			position
	`

	rows, err := s.DB.QueryContext(ctx, query, args...)
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
	stepCounts, err := s.stepCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].Assignees = assigneesByTask[tasks[i].ID]
		tasks[i].StepsDone, tasks[i].StepsTotal = stepCounts[tasks[i].ID].done, stepCounts[tasks[i].ID].total
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
