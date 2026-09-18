package tasks

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// TakenError is Take's conflict response (SPEC gate 2.36): someone was
// already on the job by the time this transaction ran, so nothing
// changed. Names is who's on it now — in practice always the single
// person who won the race, since Up for grabs only ever lists jobs
// with zero assignees to begin with.
type TakenError struct {
	Names []string
}

func (e *TakenError) Error() string {
	return "already taken by " + strings.Join(e.Names, ", ")
}

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

	names, err := assigneeNamesTx(ctx, tx, taskID)
	if err != nil {
		return err
	}
	if len(names) > 0 {
		return &TakenError{Names: names}
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

// assigneeNamesTx returns a task's current assignees' names, in the
// same deterministic (person id) order assigneesFor uses, from inside
// an in-flight transaction — Take needs the exact names for gate
// 2.36's conflict message before it rolls back.
func assigneeNamesTx(ctx context.Context, tx *sql.Tx, taskID int64) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT p.name FROM task_assignees ta JOIN people p ON p.id = ta.person_id
		WHERE ta.task_id = ? ORDER BY p.id
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
