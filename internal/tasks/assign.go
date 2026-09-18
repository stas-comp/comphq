package tasks

import (
	"context"
	"errors"
	"time"
)

// ErrInvalidAssign is returned when neither from_person nor to_person is
// given — there'd be nothing to do.
var ErrInvalidAssign = errors.New("at least one of from_person or to_person is required")

// Assign moves one person off a task and/or one person onto it,
// keeping every other assignee untouched, and never changing the
// task's stage or position (SPEC B4/gate 2.28): "Dragging a job from
// one person to another takes the first person off it and puts the
// second on, and anyone else on the job stays." fromPersonID/
// toPersonID of 0 means "not given" — this is also how "Give back"
// (P2-10, from_person = self, to_person = 0) and a plain Unassigned→
// person drag (from_person = 0) reuse the same endpoint.
func (s *Store) Assign(ctx context.Context, taskID int64, fromPersonID, toPersonID int64, actorID int64, now time.Time) error {
	if fromPersonID == 0 && toPersonID == 0 {
		return ErrInvalidAssign
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE id = ? AND removed_at IS NULL`, taskID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return ErrTaskNotFound
	}

	nowStr := now.UTC().Format(time.RFC3339)

	if fromPersonID != 0 {
		res, err := tx.ExecContext(ctx, `DELETE FROM task_assignees WHERE task_id = ? AND person_id = ?`, taskID, fromPersonID)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil {
			return err
		} else if n > 0 {
			if err := recordActivity(ctx, tx, taskID, actorID, "unassigned", personDetail{PersonID: fromPersonID}, nowStr); err != nil {
				return err
			}
		}
	}

	if toPersonID != 0 {
		var already int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_assignees WHERE task_id = ? AND person_id = ?`, taskID, toPersonID).Scan(&already); err != nil {
			return err
		}
		if already == 0 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO task_assignees (task_id, person_id) VALUES (?, ?)`, taskID, toPersonID); err != nil {
				return err
			}
			if err := recordActivity(ctx, tx, taskID, actorID, "assigned", personDetail{PersonID: toPersonID}, nowStr); err != nil {
				return err
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET updated_by = ?, updated_at = ? WHERE id = ?`, actorID, nowStr, taskID); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

