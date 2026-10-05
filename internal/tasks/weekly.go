package tasks

import (
	"context"
	"database/sql"
	"time"
)

// Weekly jobs (SPEC B13.5, D-95, gates 7.30-7.39). A weekly job is one card
// that goes back to To do each Saturday: it is never copied, so a missed week
// stays visible as an unfinished, overdue job.

const (
	repeatNone   = "none"
	repeatWeekly = "weekly"
	dateLayout   = "2006-01-02"
)

// saturdayOnOrBefore is the Saturday that began the week t is in: t itself on
// a Saturday.
func saturdayOnOrBefore(t time.Time) time.Time {
	return t.AddDate(0, 0, -((int(t.Weekday()) + 1) % 7))
}

// saturdayOnOrAfter is the Briefing's Saturday (A8): t itself on a Saturday,
// otherwise the coming one.
func saturdayOnOrAfter(t time.Time) time.Time {
	return t.AddDate(0, 0, (int(time.Saturday)-int(t.Weekday())+7)%7)
}

// weeklyResetDetail is what a "weekly_reset" History row remembers. Nobody is
// recorded as having done it (task_activity.person_id must be a real person,
// and a made-up "Comp HQ" person would show up in name lists, the export and
// the picker); it is filed under the job's creator, marked by "by".
type weeklyResetDetail struct {
	By  string `json:"by"`
	For string `json:"for"`
}

// setWeeklyTx turns a job's weekly repeat on or off inside the caller's
// transaction (gates 7.30, 7.31, 7.37). Turning it on starts it at the
// Saturday on or before today, so it does not reset straight away, and gives
// a job with no due date the Briefing's Saturday. Turning it off makes it an
// ordinary job from then on. Nothing happens when it is already as asked.
func setWeeklyTx(ctx context.Context, tx *sql.Tx, taskID int64, on bool, actorID int64, now time.Time) error {
	var current string
	var due sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT repeat, due_date FROM tasks WHERE id = ?`, taskID).Scan(&current, &due); err != nil {
		return err
	}
	nowStr := now.UTC().Format(time.RFC3339)
	switch {
	case on && current != repeatWeekly:
		start := saturdayOnOrBefore(now).Format(dateLayout)
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET repeat = ?, repeat_reset_on = ? WHERE id = ?`, repeatWeekly, start, taskID); err != nil {
			return err
		}
		if !due.Valid || due.String == "" {
			thisSaturday := saturdayOnOrAfter(now).Format(dateLayout)
			if _, err := tx.ExecContext(ctx, `UPDATE tasks SET due_date = ? WHERE id = ?`, thisSaturday, taskID); err != nil {
				return err
			}
			if err := recordActivity(ctx, tx, taskID, actorID, "due_changed", dueChangedDetail{DueDate: thisSaturday}, nowStr); err != nil {
				return err
			}
		}
		return recordActivity(ctx, tx, taskID, actorID, "repeat_on", nil, nowStr)
	case !on && current == repeatWeekly:
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET repeat = ? WHERE id = ?`, repeatNone, taskID); err != nil {
			return err
		}
		return recordActivity(ctx, tx, taskID, actorID, "repeat_off", nil, nowStr)
	}
	return nil
}

// ResetWeekly puts every weekly job back for the Saturday on or before today
// (gates 7.32-7.36). In one transaction, for every unremoved weekly job whose
// last reset was before that Saturday:
//   - one that is Done goes to the bottom of To do, due that Saturday, with
//     the same people and its steps unticked (ticks are not History, D-71),
//     and a "weekly_reset" History row;
//   - one that is not Done stays exactly where it is, with its old due date,
//     so it shows OVERDUE (gate 7.33): nothing is ever copied;
//   - either way its reset date moves to that Saturday, which is why a reset
//     happens once, however many Saturdays were missed (gate 7.35), and why
//     a second run on the same Saturday finds nothing to do.
//
// There is no clock job: this runs at start-up and at the top of every
// request that shows jobs, so a NAS that was off on a Saturday catches up the
// first time anyone opens Comp HQ, and test mode's fixed date works. The case
// with nothing to do is one cheap query. It returns how many jobs went back to
// To do; the change counter moves only when that is more than none.
func (s *Store) ResetWeekly(ctx context.Context, today time.Time) (int, error) {
	saturday := saturdayOnOrBefore(today).Format(dateLayout)

	var one int
	err := s.DB.QueryRowContext(ctx,
		`SELECT 1 FROM tasks WHERE repeat = ? AND removed_at IS NULL AND (repeat_reset_on IS NULL OR repeat_reset_on < ?) LIMIT 1`,
		repeatWeekly, saturday,
	).Scan(&one)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Re-check inside the transaction: if two requests raced, the second finds
	// the first one's work already marked and has nothing to do.
	rows, err := tx.QueryContext(ctx,
		`SELECT id, stage, created_by FROM tasks
		 WHERE repeat = ? AND removed_at IS NULL AND (repeat_reset_on IS NULL OR repeat_reset_on < ?)
		 ORDER BY position, id`,
		repeatWeekly, saturday,
	)
	if err != nil {
		return 0, err
	}
	type job struct {
		id, createdBy int64
		stage         string
	}
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.stage, &j.createdBy); err != nil {
			rows.Close()
			return 0, err
		}
		jobs = append(jobs, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(jobs) == 0 {
		return 0, nil
	}

	nowStr := today.UTC().Format(time.RFC3339)
	reset := 0
	for _, j := range jobs {
		if j.stage == StageDone {
			todo, err := stageOrder(ctx, tx, StageTodo, j.id)
			if err != nil {
				return 0, err
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE tasks SET stage = ?, position = ?, done_at = NULL, due_date = ?, updated_by = created_by, updated_at = ? WHERE id = ?`,
				StageTodo, len(todo)+1, saturday, nowStr, j.id,
			); err != nil {
				return 0, err
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE task_checklist_items SET done_by = NULL, done_at = NULL WHERE task_id = ? AND removed_at IS NULL AND done_at IS NOT NULL`,
				j.id,
			); err != nil {
				return 0, err
			}
			if err := recordActivity(ctx, tx, j.id, j.createdBy, "weekly_reset", weeklyResetDetail{By: "comphq", For: saturday}, nowStr); err != nil {
				return 0, err
			}
			reset++
		}
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET repeat_reset_on = ? WHERE id = ?`, saturday, j.id); err != nil {
			return 0, err
		}
	}
	if reset > 0 {
		doneOrder, err := stageOrder(ctx, tx, StageDone, 0)
		if err != nil {
			return 0, err
		}
		if err := renumber(ctx, tx, doneOrder); err != nil {
			return 0, err
		}
		if err := bumpTasksVersion(ctx, tx); err != nil {
			return 0, err
		}
	}
	return reset, tx.Commit()
}
