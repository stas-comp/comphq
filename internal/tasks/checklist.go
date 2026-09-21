package tasks

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/stas-comp/comphq/internal/app/format"
)

// Steps inside a job (SPEC B10, B4 "Checklist steps"). One flat list per
// task; a step is a tick-box with words on it. Every write here bumps the
// tasks change counter in its own transaction (gate 5.12), and both
// limits are enforced in this file so no route can get round them.
const (
	// MaxSteps is how many non-removed steps a job can hold (gate 5.13).
	MaxSteps = 50
	// MaxStepChars is how long a step's words can be (gate 5.13).
	MaxStepChars = 200
)

var (
	ErrEmptyStep    = errors.New("a step needs some words")
	ErrStepTooLong  = errors.New("a step can be up to 200 characters")
	ErrTooManySteps = errors.New("a job can have up to 50 steps")
	// ErrStepNotFound is a step id that doesn't exist on this task at all.
	ErrStepNotFound = errors.New("that step isn't there")
	// ErrStepRemoved is acting on a step somebody else has already removed
	// (gate 5.12): the caller shows "Somebody else removed that step" and
	// nothing changes.
	ErrStepRemoved = errors.New("that step has been removed")
	// ErrInvalidDirection is a move that isn't "up" or "down".
	ErrInvalidDirection = errors.New("a step moves up or down")
)

// Directions a step can be moved one place in (gate 5.07).
const (
	MoveUp   = "up"
	MoveDown = "down"
)

// Step is one row of task_checklist_items, with who ticked it resolved.
type Step struct {
	ID       int64
	TaskID   int64
	Text     string
	Position int
	Done     bool
	Removed  bool
	// DoneByName and DoneDate are set only when Done: who ticked it and
	// when, in the short form used on cards ("Jane", "Sat 19 Sep").
	DoneByName string
	DoneDate   string
	DoneAt     time.Time
	// RemovedAt is set only on a removed step, which only the export lists
	// (gate 5.14).
	RemovedAt time.Time
}

// Detail payload for the step_* task_activity rows (SPEC B3: "short JSON").
// Ticking is deliberately never recorded (D-71).
type stepDetail struct {
	StepID int64  `json:"step_id"`
	Text   string `json:"text"`
	// From is the old words of a rename.
	From string `json:"from,omitempty"`
}

// cleanStepText tidies typed words into one line and checks the limits
// that depend only on the words themselves.
func cleanStepText(text string) (string, error) {
	text = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(text)
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ErrEmptyStep
	}
	if utf8.RuneCountInString(text) > MaxStepChars {
		return "", ErrStepTooLong
	}
	return text, nil
}

// stepRow is the part of a step the write paths need to look at.
type stepRow struct {
	text     string
	position int
	done     bool
	removed  bool
}

// loadStep reads one step of one task inside tx. A step id that belongs to
// another task is treated as not found.
func loadStep(ctx context.Context, tx *sql.Tx, taskID, stepID int64) (stepRow, error) {
	var r stepRow
	var doneAt, removedAt sql.NullString
	err := tx.QueryRowContext(ctx,
		`SELECT text, position, done_at, removed_at FROM task_checklist_items WHERE id = ? AND task_id = ?`,
		stepID, taskID,
	).Scan(&r.text, &r.position, &doneAt, &removedAt)
	if err == sql.ErrNoRows {
		return stepRow{}, ErrStepNotFound
	}
	if err != nil {
		return stepRow{}, err
	}
	r.done = doneAt.Valid
	r.removed = removedAt.Valid
	return r, nil
}

// activeStepIDs returns a task's non-removed step ids in display order.
func activeStepIDs(ctx context.Context, tx *sql.Tx, taskID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id FROM task_checklist_items WHERE task_id = ? AND removed_at IS NULL ORDER BY position, id`,
		taskID,
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

// renumberSteps writes positions 1..n down an ordered id list, the way
// renumber does for a stage's tasks.
func renumberSteps(ctx context.Context, tx *sql.Tx, ids []int64) error {
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx,
			`UPDATE task_checklist_items SET position = ? WHERE id = ? AND position <> ?`,
			i+1, id, i+1,
		); err != nil {
			return err
		}
	}
	return nil
}

// ListSteps returns a task's non-removed steps in order. today is only
// used to choose the short form of a tick's date.
func (s *Store) ListSteps(ctx context.Context, taskID int64, today time.Time) ([]Step, error) {
	return s.listSteps(ctx, taskID, false, today)
}

// ListStepsWithRemoved is ListSteps plus the removed steps, listed after
// the live ones. Only the export uses it (gate 5.14: a removed step still
// appears in steps.csv).
func (s *Store) ListStepsWithRemoved(ctx context.Context, taskID int64, today time.Time) ([]Step, error) {
	return s.listSteps(ctx, taskID, true, today)
}

func (s *Store) listSteps(ctx context.Context, taskID int64, withRemoved bool, today time.Time) ([]Step, error) {
	where := `c.task_id = ? AND c.removed_at IS NULL`
	order := `c.position, c.id`
	if withRemoved {
		where = `c.task_id = ?`
		order = `(c.removed_at IS NOT NULL), c.position, c.id`
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.id, c.task_id, c.text, c.position, c.done_by, COALESCE(p.name, ''), c.done_at, c.removed_at
		FROM task_checklist_items c
		LEFT JOIN people p ON p.id = c.done_by
		WHERE `+where+`
		ORDER BY `+order, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []Step
	for rows.Next() {
		var st Step
		var doneBy sql.NullInt64
		var doneAt, removedAt sql.NullString
		if err := rows.Scan(&st.ID, &st.TaskID, &st.Text, &st.Position, &doneBy, &st.DoneByName, &doneAt, &removedAt); err != nil {
			return nil, err
		}
		if doneAt.Valid {
			t, err := time.Parse(time.RFC3339, doneAt.String)
			if err != nil {
				return nil, err
			}
			t = t.In(today.Location())
			st.Done = true
			st.DoneAt = t
			st.DoneDate = format.DateShort(t, today)
		}
		if removedAt.Valid {
			t, err := time.Parse(time.RFC3339, removedAt.String)
			if err != nil {
				return nil, err
			}
			st.Removed = true
			st.RemovedAt = t.In(today.Location())
		}
		steps = append(steps, st)
	}
	return steps, rows.Err()
}

// AddStep adds a step at the end of a job's list and records "step_added".
func (s *Store) AddStep(ctx context.Context, taskID int64, text string, actorID int64, now time.Time) (Step, error) {
	text, err := cleanStepText(text)
	if err != nil {
		return Step{}, err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Step{}, err
	}
	defer tx.Rollback()

	var one int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM tasks WHERE id = ? AND removed_at IS NULL`, taskID).Scan(&one)
	if err == sql.ErrNoRows {
		return Step{}, ErrTaskNotFound
	}
	if err != nil {
		return Step{}, err
	}

	var count int
	var maxPos sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*), MAX(position) FROM task_checklist_items WHERE task_id = ? AND removed_at IS NULL`, taskID,
	).Scan(&count, &maxPos); err != nil {
		return Step{}, err
	}
	if count >= MaxSteps {
		return Step{}, ErrTooManySteps
	}

	nowStr := now.UTC().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx,
		`INSERT INTO task_checklist_items (task_id, text, position, created_by, created_at, updated_by, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		taskID, text, int(maxPos.Int64)+1, actorID, nowStr, actorID, nowStr,
	)
	if err != nil {
		return Step{}, err
	}
	stepID, err := res.LastInsertId()
	if err != nil {
		return Step{}, err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "step_added", stepDetail{StepID: stepID, Text: text}, nowStr); err != nil {
		return Step{}, err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return Step{}, err
	}
	if err := tx.Commit(); err != nil {
		return Step{}, err
	}
	return s.getStep(ctx, taskID, stepID, now)
}

func (s *Store) getStep(ctx context.Context, taskID, stepID int64, today time.Time) (Step, error) {
	steps, err := s.listSteps(ctx, taskID, true, today)
	if err != nil {
		return Step{}, err
	}
	for _, st := range steps {
		if st.ID == stepID {
			return st, nil
		}
	}
	return Step{}, ErrStepNotFound
}

// SetStepDone ticks or unticks a step. Ticking a step somebody else has
// already ticked is not an error: it stays ticked and keeps the first
// person's name and time (gate 5.12). Unticking clears both. Neither is
// recorded in History (D-71). A removed step answers ErrStepRemoved.
func (s *Store) SetStepDone(ctx context.Context, taskID, stepID int64, done bool, actorID int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row, err := loadStep(ctx, tx, taskID, stepID)
	if err != nil {
		return err
	}
	if row.removed {
		return ErrStepRemoved
	}
	if row.done == done {
		return nil
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if done {
		_, err = tx.ExecContext(ctx,
			`UPDATE task_checklist_items SET done_by = ?, done_at = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
			actorID, nowStr, actorID, nowStr, stepID)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE task_checklist_items SET done_by = NULL, done_at = NULL, updated_by = ?, updated_at = ? WHERE id = ?`,
			actorID, nowStr, stepID)
	}
	if err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RenameStep changes a step's words in place and records "step_renamed".
// Saving the same words again changes nothing.
func (s *Store) RenameStep(ctx context.Context, taskID, stepID int64, text string, actorID int64, now time.Time) error {
	text, err := cleanStepText(text)
	if err != nil {
		return err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row, err := loadStep(ctx, tx, taskID, stepID)
	if err != nil {
		return err
	}
	if row.removed {
		return ErrStepRemoved
	}
	if row.text == text {
		return nil
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE task_checklist_items SET text = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		text, actorID, nowStr, stepID,
	); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "step_renamed", stepDetail{StepID: stepID, Text: text, From: row.text}, nowStr); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveStep soft-deletes a step (SPEC A3: nothing is ever permanently
// deleted) and closes the gap in the remaining positions. The removed row
// keeps its own position, which is how RestoreStep knows where it was.
func (s *Store) RemoveStep(ctx context.Context, taskID, stepID int64, actorID int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row, err := loadStep(ctx, tx, taskID, stepID)
	if err != nil {
		return err
	}
	if row.removed {
		return ErrStepRemoved
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE task_checklist_items SET removed_at = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		nowStr, actorID, nowStr, stepID,
	); err != nil {
		return err
	}
	remaining, err := activeStepIDs(ctx, tx, taskID)
	if err != nil {
		return err
	}
	if err := renumberSteps(ctx, tx, remaining); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "step_removed", stepDetail{StepID: stepID, Text: row.text}, nowStr); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// RestoreStep is the Undo: it puts a removed step back at its old
// position, pushing later steps down if that position has since been
// taken (SPEC B4). Restoring a step that is already back changes nothing.
func (s *Store) RestoreStep(ctx context.Context, taskID, stepID int64, actorID int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row, err := loadStep(ctx, tx, taskID, stepID)
	if err != nil {
		return err
	}
	if !row.removed {
		return nil
	}

	ids, err := activeStepIDs(ctx, tx, taskID)
	if err != nil {
		return err
	}
	if len(ids) >= MaxSteps {
		return ErrTooManySteps
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE task_checklist_items SET removed_at = NULL, updated_by = ?, updated_at = ? WHERE id = ?`,
		actorID, nowStr, stepID,
	); err != nil {
		return err
	}
	at := row.position - 1
	if at < 0 {
		at = 0
	}
	if at > len(ids) {
		at = len(ids)
	}
	ordered := make([]int64, 0, len(ids)+1)
	ordered = append(ordered, ids[:at]...)
	ordered = append(ordered, stepID)
	ordered = append(ordered, ids[at:]...)
	if err := renumberSteps(ctx, tx, ordered); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "step_restored", stepDetail{StepID: stepID, Text: row.text}, nowStr); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// MoveStep moves a step one place up or down. A step already at that end
// of the list stays where it is; that is not an error.
func (s *Store) MoveStep(ctx context.Context, taskID, stepID int64, direction string, actorID int64, now time.Time) error {
	if direction != MoveUp && direction != MoveDown {
		return ErrInvalidDirection
	}
	return s.reorderStep(ctx, taskID, stepID, actorID, now, func(index int) int {
		if direction == MoveUp {
			return index - 1
		}
		return index + 1
	})
}

// MoveStepTo puts a step at a 1-based position in the list, which is how
// dragging reorders (gate 5.07). A position past either end is the end.
func (s *Store) MoveStepTo(ctx context.Context, taskID, stepID int64, position int, actorID int64, now time.Time) error {
	return s.reorderStep(ctx, taskID, stepID, actorID, now, func(index int) int {
		return position - 1
	})
}

// reorderStep moves a step to whatever index target picks, clamped to the
// list, and writes nothing (and bumps nothing) if that is where it
// already is.
func (s *Store) reorderStep(ctx context.Context, taskID, stepID, actorID int64, now time.Time, target func(index int) int) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row, err := loadStep(ctx, tx, taskID, stepID)
	if err != nil {
		return err
	}
	if row.removed {
		return ErrStepRemoved
	}
	ids, err := activeStepIDs(ctx, tx, taskID)
	if err != nil {
		return err
	}
	index := -1
	for i, id := range ids {
		if id == stepID {
			index = i
		}
	}
	if index < 0 {
		return ErrStepNotFound
	}
	to := target(index)
	if to < 0 {
		to = 0
	}
	if to > len(ids)-1 {
		to = len(ids) - 1
	}
	if to == index {
		return nil
	}

	rest := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != stepID {
			rest = append(rest, id)
		}
	}
	ordered := make([]int64, 0, len(ids))
	ordered = append(ordered, rest[:to]...)
	ordered = append(ordered, stepID)
	ordered = append(ordered, rest[to:]...)
	if err := renumberSteps(ctx, tx, ordered); err != nil {
		return err
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE task_checklist_items SET updated_by = ?, updated_at = ? WHERE id = ?`,
		actorID, nowStr, stepID,
	); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
