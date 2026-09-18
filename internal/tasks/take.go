package tasks

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ErrTaskAlreadyTaken is returned by Take when someone is already on
// the task. SPEC gate 2.36's exact conflict response (409 with the
// assignees' names) is P2-11's own job — this only guarantees Take
// never silently double-assigns a job two people went for at once.
var ErrTaskAlreadyTaken = errors.New("someone has already taken this task")

// Take adds the current person to an unassigned task (SPEC B4): an
// idea becomes a todo task, placed at the given drop position or the
// bottom (gate 2.35); a todo/doing task keeps its own position unless
// a drop position is given (gate 2.34). Records activity for the
// assignment, and for the stage change too when there is one.
func (s *Store) Take(ctx context.Context, taskID int64, beforeID, afterID int64, actorID int64, now time.Time) error {
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

	var assigneeCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_assignees WHERE task_id = ?`, taskID).Scan(&assigneeCount); err != nil {
		return err
	}
	if assigneeCount > 0 {
		return ErrTaskAlreadyTaken
	}

	nowStr := now.UTC().Format(time.RFC3339)
	newStage := stage
	if stage == StageIdea {
		newStage = StageTodo
	}

	hasDropPosition := beforeID != 0 || afterID != 0
	if newStage != stage || hasDropPosition {
		targetOthers, err := stageOrder(ctx, tx, newStage, taskID)
		if err != nil {
			return err
		}
		var newOrder []int64
		if hasDropPosition {
			newOrder, err = insertTask(targetOthers, taskID, beforeID, afterID, false)
			if err != nil {
				return err
			}
		} else {
			newOrder = append(targetOthers, taskID)
		}
		if err := renumber(ctx, tx, newOrder); err != nil {
			return err
		}

		if newStage != stage {
			oldOthers, err := stageOrder(ctx, tx, stage, taskID)
			if err != nil {
				return err
			}
			if err := renumber(ctx, tx, oldOthers); err != nil {
				return err
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO task_assignees (task_id, person_id) VALUES (?, ?)`, taskID, actorID); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "assigned", personDetail{PersonID: actorID}, nowStr); err != nil {
		return err
	}

	if newStage != stage {
		if _, err := tx.ExecContext(ctx,
			`UPDATE tasks SET stage = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
			newStage, actorID, nowStr, taskID,
		); err != nil {
			return err
		}
		if err := updateDoneAtForStageChange(ctx, tx, taskID, stage, newStage, nowStr); err != nil {
			return err
		}
		if err := recordActivity(ctx, tx, taskID, actorID, "moved", movedDetail{From: stage, To: newStage}, nowStr); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET updated_by = ?, updated_at = ? WHERE id = ?`, actorID, nowStr, taskID); err != nil {
			return err
		}
	}

	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
