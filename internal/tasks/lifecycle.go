package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// ListFinished returns Done tasks that have aged off the board (SPEC
// gate 2.08's 14-day retention), newest-finished first.
func (s *Store) ListFinished(ctx context.Context, today time.Time) ([]Task, error) {
	cutoff := today.AddDate(0, 0, -doneRetentionDays).UTC().Format(time.RFC3339)
	return s.listTasks(ctx, `
		SELECT id, title, notes, size, stage, position, due_date, done_at
		FROM tasks
		WHERE removed_at IS NULL AND stage = 'done' AND done_at IS NOT NULL AND done_at < ?
		  AND repeat <> 'weekly'
		ORDER BY done_at DESC
	`, cutoff)
}

// ListRemoved returns every removed task (SPEC gate 2.09), newest-
// removed first. Nothing is ever permanently deleted, so this is the
// only way a removed task's data is reachable again, via Restore.
func (s *Store) ListRemoved(ctx context.Context) ([]Task, error) {
	return s.listTasks(ctx, `
		SELECT id, title, notes, size, stage, position, due_date, done_at
		FROM tasks
		WHERE removed_at IS NOT NULL
		ORDER BY removed_at DESC
	`)
}

// listTasks runs a tasks query (used by ListFinished/ListRemoved, whose
// column list and scanning are identical to ListBoard's) and resolves
// assignees for the result, ignoring Overdue — neither list is a board,
// so gate 2.03's OVERDUE stamp doesn't apply to either.
func (s *Store) listTasks(ctx context.Context, query string, args ...any) ([]Task, error) {
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
	weekly, err := s.weeklyIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].Assignees = assigneesByTask[tasks[i].ID]
		tasks[i].StepsDone, tasks[i].StepsTotal = stepCounts[tasks[i].ID].done, stepCounts[tasks[i].ID].total
		tasks[i].Weekly = weekly[tasks[i].ID]
	}
	return tasks, nil
}

// Reopen returns a finished task to the bottom of To do, clearing
// done_at and recording a "reopened" activity (SPEC gate 2.08) — the
// same position/done_at bookkeeping Move already does for an ordinary
// stage change, just with a fixed destination and a distinct activity
// action.
func (s *Store) Reopen(ctx context.Context, taskID int64, actorID int64, now time.Time) error {
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

	newOrder, err := stageOrder(ctx, tx, StageTodo, taskID)
	if err != nil {
		return err
	}
	newOrder = append(newOrder, taskID)
	if err := renumber(ctx, tx, newOrder); err != nil {
		return err
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET stage = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		StageTodo, actorID, nowStr, taskID,
	); err != nil {
		return err
	}
	if err := updateDoneAtForStageChange(ctx, tx, taskID, oldStage, StageTodo, nowStr); err != nil {
		return err
	}

	if oldStage != StageTodo {
		oldOrder, err := stageOrder(ctx, tx, oldStage, taskID)
		if err != nil {
			return err
		}
		if err := renumber(ctx, tx, oldOrder); err != nil {
			return err
		}
	}

	if err := recordActivity(ctx, tx, taskID, actorID, "reopened", nil, nowStr); err != nil {
		return err
	}

	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// Remove hides a task from the board (and, later, the calendar) without
// deleting it (SPEC gate 2.09: "Nothing can be permanently deleted").
// The stage it leaves is renumbered so position stays a contiguous
// 1..n among what's still visible there.
func (s *Store) Remove(ctx context.Context, taskID int64, actorID int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var stage string
	err = tx.QueryRowContext(ctx, `SELECT stage FROM tasks WHERE id = ? AND removed_at IS NULL`, taskID).Scan(&stage)
	if err == sql.ErrNoRows {
		return ErrTaskNotFound
	}
	if err != nil {
		return err
	}

	// Remember which job it sat beside, so Undo can put it back exactly there
	// (gate 7.16, D-93): the job that followed it, or the one before it when
	// it was last. Only new removals carry this; older ones fall back to the
	// row's old position.
	before, err := stageOrder(ctx, tx, stage, 0)
	if err != nil {
		return err
	}
	detail := removedDetail{}
	for i, id := range before {
		if id != taskID {
			continue
		}
		if i+1 < len(before) {
			detail.BeforeID = before[i+1]
		} else if i > 0 {
			detail.AfterID = before[i-1]
		}
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET removed_at = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		nowStr, actorID, nowStr, taskID,
	); err != nil {
		return err
	}

	remaining, err := stageOrder(ctx, tx, stage, taskID)
	if err != nil {
		return err
	}
	if err := renumber(ctx, tx, remaining); err != nil {
		return err
	}

	if err := recordActivity(ctx, tx, taskID, actorID, "removed", detail, nowStr); err != nil {
		return err
	}

	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// removedDetail is what a "removed" activity row remembers: the job that was
// next to the removed one (gate 7.16). Both are 0 when the stage held only
// this job.
type removedDetail struct {
	BeforeID int64 `json:"before_id,omitempty"`
	AfterID  int64 `json:"after_id,omitempty"`
}

// Undo puts a job someone has just removed back where it was (gates 7.15,
// 7.16, D-93): in its column, directly before the job that followed it when
// it was removed (or after the one that preceded it). If that neighbour has
// since moved to another column or been removed itself, it goes back to its
// old position, held within the column's length. A job that is not removed
// (someone else already restored it, or undid it) is left alone, with no
// error: restored reports whether it was removed to begin with. People,
// steps and links were never touched by removal, so they are all still there.
// Restore (Removed tasks) is a different thing and still goes to the bottom.
func (s *Store) Undo(ctx context.Context, taskID int64, actorID int64, now time.Time) (restored bool, err error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var stage string
	var oldPosition int
	err = tx.QueryRowContext(ctx,
		`SELECT stage, position FROM tasks WHERE id = ? AND removed_at IS NOT NULL`, taskID,
	).Scan(&stage, &oldPosition)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var detailJSON string
	err = tx.QueryRowContext(ctx,
		`SELECT detail FROM task_activity WHERE task_id = ? AND action = 'removed' ORDER BY id DESC LIMIT 1`, taskID,
	).Scan(&detailJSON)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	var detail removedDetail
	if detailJSON != "" {
		_ = json.Unmarshal([]byte(detailJSON), &detail) // an older row has "{}": fall back below
	}

	order, err := stageOrder(ctx, tx, stage, taskID)
	if err != nil {
		return false, err
	}
	index := -1
	for i, id := range order {
		if detail.BeforeID != 0 && id == detail.BeforeID {
			index = i
			break
		}
		if detail.AfterID != 0 && id == detail.AfterID {
			index = i + 1
			break
		}
	}
	if index < 0 {
		index = oldPosition - 1
		if index < 0 {
			index = 0
		}
		if index > len(order) {
			index = len(order)
		}
	}
	newOrder := make([]int64, 0, len(order)+1)
	newOrder = append(newOrder, order[:index]...)
	newOrder = append(newOrder, taskID)
	newOrder = append(newOrder, order[index:]...)
	if err := renumber(ctx, tx, newOrder); err != nil {
		return false, err
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET removed_at = NULL, updated_by = ?, updated_at = ? WHERE id = ?`,
		actorID, nowStr, taskID,
	); err != nil {
		return false, err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "undone", nil, nowStr); err != nil {
		return false, err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// RemovedTitle is the title of a task that is currently removed, for the
// message that offers Undo; ok is false for a task that isn't removed.
func (s *Store) RemovedTitle(ctx context.Context, taskID int64) (title string, ok bool, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT title FROM tasks WHERE id = ? AND removed_at IS NOT NULL`, taskID).Scan(&title)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return title, err == nil, err
}

// Restore brings a removed task back, landing at the bottom of the
// stage it was in when removed (SPEC gate 2.09) — not its old position,
// which may since have been reassigned to another task by Remove's own
// renumbering.
func (s *Store) Restore(ctx context.Context, taskID int64, actorID int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var stage string
	err = tx.QueryRowContext(ctx, `SELECT stage FROM tasks WHERE id = ? AND removed_at IS NOT NULL`, taskID).Scan(&stage)
	if err == sql.ErrNoRows {
		return ErrTaskNotFound
	}
	if err != nil {
		return err
	}

	newOrder, err := stageOrder(ctx, tx, stage, taskID)
	if err != nil {
		return err
	}
	newOrder = append(newOrder, taskID)
	if err := renumber(ctx, tx, newOrder); err != nil {
		return err
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET removed_at = NULL, updated_by = ?, updated_at = ? WHERE id = ?`,
		actorID, nowStr, taskID,
	); err != nil {
		return err
	}

	if err := recordActivity(ctx, tx, taskID, actorID, "restored", nil, nowStr); err != nil {
		return err
	}

	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
