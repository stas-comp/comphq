package tasks

import (
	"context"
	"database/sql"
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
	for i := range tasks {
		tasks[i].Assignees = assigneesByTask[tasks[i].ID]
		tasks[i].StepsDone, tasks[i].StepsTotal = stepCounts[tasks[i].ID].done, stepCounts[tasks[i].ID].total
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

	if err := recordActivity(ctx, tx, taskID, actorID, "removed", nil, nowStr); err != nil {
		return err
	}

	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
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
