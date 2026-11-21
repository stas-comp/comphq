package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Yearly jobs (SPEC B14.4, D-108, gates 8.40-8.53). A yearly job is one job that
// waits out of sight, shows in To do a chosen number of weeks before it is due,
// and, once it has been Done for 14 days, is put away until next year.
//
// WAITING is computed: repeat = 'yearly' AND shows_from > today. There is no
// stored flag and no clock job, so test mode's fixed date works. Every list that
// shows jobs leaves a waiting job out through waitingSQL, the one place that says
// what waiting means, and the order code works only over jobs that aren't waiting
// (stageOrder), so a waiting job can never move anything.

const (
	repeatYearly       = "yearly"
	defaultNoticeWeeks = 6
	minNoticeWeeks     = 1
	maxNoticeWeeks     = 52
)

var (
	// ErrYearlyNeedsDue is the refusal for a yearly job with no due date (gate 8.41).
	ErrYearlyNeedsDue = errors.New("A yearly job needs a due date.")
	// ErrNoticeWeeks is a notice outside 1 to 52 weeks.
	ErrNoticeWeeks = errors.New("The notice has to be between 1 and 52 weeks.")
)

// waitingClause is "this job is not waiting", for a query on tasks with the
// given alias ("" for none). It takes one argument: today as a local date. A
// job with no show date is never waiting (NULL > today is not true).
func waitingClause(alias string) string {
	if alias != "" {
		alias += "."
	}
	return "NOT (" + alias + "repeat = 'yearly' AND COALESCE(" + alias + "shows_from, '') > ?)"
}

// dateOf is the local date a time is on, the form dates are stored in.
func dateOf(t time.Time) string { return t.Format(dateLayout) }

// noticeDaysFor checks and converts weeks of notice (0 means the default).
func noticeDaysFor(weeks int) (int, error) {
	if weeks == 0 {
		weeks = defaultNoticeWeeks
	}
	if weeks < minNoticeWeeks || weeks > maxNoticeWeeks {
		return 0, ErrNoticeWeeks
	}
	return weeks * 7, nil
}

// showsFromFor is the due date minus the notice.
func showsFromFor(due string, noticeDays int) (string, error) {
	d, err := time.ParseInLocation(dateLayout, due, time.UTC)
	if err != nil {
		return "", err
	}
	return d.AddDate(0, 0, -noticeDays).Format(dateLayout), nil
}

// nextYearlyDue is the date a yearly job is next due after oldDue: the anchor's
// month and day in the year after oldDue's, with 29 February clamped to 28
// February in a year that has no 29th (gate 8.45). The anchor keeps 29 February,
// so a leap year brings it back.
func nextYearlyDue(oldDue, anchor string) string {
	d, err := time.ParseInLocation(dateLayout, oldDue, time.UTC)
	if err != nil {
		return oldDue
	}
	year := d.Year() + 1
	var month, dom int
	if _, err := fmt.Sscanf(anchor, "%d-%d", &month, &dom); err != nil || month < 1 || month > 12 || dom < 1 {
		month, dom = int(d.Month()), d.Day()
	}
	if month == 2 && dom == 29 && !isLeap(year) {
		dom = 28
	}
	return time.Date(year, time.Month(month), dom, 0, 0, 0, 0, time.UTC).Format(dateLayout)
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// yearlyDetail is what the yearly History rows remember. Nobody is recorded as
// having done them (as weekly_reset): they are filed under the job's creator,
// marked by "by".
type yearlyDetail struct {
	By    string `json:"by"`
	Until string `json:"until,omitempty"` // yearly_wait: the date it will appear
	Due   string `json:"due,omitempty"`   // yearly_shown: its due date
}

type repeatDetail struct {
	Kind string `json:"kind,omitempty"`
}

// setYearlyTx turns a job's yearly repeat on, or refreshes it after its due date
// or notice changed, inside the caller's transaction (gates 8.40, 8.41). It sets
// the anchor, the notice and the show date, and clears the weekly reset date so
// a weekly job's last Saturday can never be mistaken for a showing. A job whose
// show date has already come is marked as shown, so it is not "shown" a second
// time by the pass. days is the notice in days (noticeDaysFor). It reports whether the job went
// from waiting to showing, so the caller can put it at the bottom of To do.
func setYearlyTx(ctx context.Context, tx *sql.Tx, taskID int64, days int, actorID int64, now time.Time) (becameVisible bool, err error) {
	var due sql.NullString
	var current string
	var oldShows sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT due_date, repeat, shows_from FROM tasks WHERE id = ?`, taskID).Scan(&due, &current, &oldShows); err != nil {
		return false, err
	}
	if !due.Valid || due.String == "" {
		return false, ErrYearlyNeedsDue
	}
	shows, err := showsFromFor(due.String, days)
	if err != nil {
		return false, err
	}
	anchor := due.String[5:] // MM-DD
	today := dateOf(now)
	wasWaiting := current == repeatYearly && oldShows.Valid && oldShows.String > today
	var resetOn any
	if shows <= today {
		resetOn = shows // already showing: this year's showing is done
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET repeat = ?, repeat_anchor = ?, notice_days = ?, shows_from = ?, repeat_reset_on = ? WHERE id = ?`,
		repeatYearly, anchor, days, shows, resetOn, taskID); err != nil {
		return false, err
	}
	if current != repeatYearly {
		if err := recordActivity(ctx, tx, taskID, actorID, "repeat_on", repeatDetail{Kind: repeatYearly}, now.UTC().Format(time.RFC3339)); err != nil {
			return false, err
		}
	}
	return wasWaiting && shows <= today, nil
}

// clearYearlyTx makes a yearly job an ordinary one again (gate 8.40).
func clearYearlyTx(ctx context.Context, tx *sql.Tx, taskID, actorID int64, now time.Time) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET repeat = ?, repeat_anchor = NULL, notice_days = NULL, shows_from = NULL WHERE id = ?`, repeatNone, taskID); err != nil {
		return err
	}
	return recordActivity(ctx, tx, taskID, actorID, "repeat_off", repeatDetail{Kind: repeatYearly}, now.UTC().Format(time.RFC3339))
}

// ResetYearly is the yearly pass (gates 8.44-8.47). In one transaction:
//
//  1. Put away. A yearly job that has been Done for more than 14 days (the same
//     moment an ordinary job would go to Finished tasks) goes back to waiting:
//     its due date moves on one year from its OLD due date, never from the day it
//     was finished; its show date follows; it leaves Done for To do with its steps
//     unticked and its people on it; History records "yearly_wait". If the new
//     show date has already passed (a very long notice, or the NAS was off) it
//     goes straight on to step 2.
//  2. Show. A yearly job that is not Done, whose show date has come and whose
//     showing hasn't been done, goes to the bottom of To do, and its showing is
//     marked done (repeat_reset_on = shows_from) so it happens once. History
//     records "yearly_shown".
//
// An unfinished yearly job is never moved on or doubled up: it stays where it is,
// overdue, until somebody finishes it. There is no clock job: this runs at
// start-up and at the top of every request that shows jobs, so a NAS that was off
// catches up the first time anyone opens Comp HQ, once, and the case with nothing
// to do is one cheap query. It returns how many jobs were put away or shown; the
// change counter moves only when that is more than none.
func (s *Store) ResetYearly(ctx context.Context, today time.Time) (int, error) {
	day := dateOf(today)
	cutoff := today.AddDate(0, 0, -doneRetentionDays).UTC().Format(time.RFC3339)

	var one int
	err := s.DB.QueryRowContext(ctx, `
		SELECT 1 FROM tasks
		WHERE repeat = ? AND removed_at IS NULL AND (
			(stage = 'done' AND done_at IS NOT NULL AND done_at < ?)
			OR (stage != 'done' AND shows_from <= ? AND (repeat_reset_on IS NULL OR repeat_reset_on != shows_from))
		) LIMIT 1`, repeatYearly, cutoff, day).Scan(&one)
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

	nowStr := today.UTC().Format(time.RFC3339)
	changed := map[int64]bool{} // jobs put away or shown (one that is both counts once)

	// 1. Put away: re-checked inside the transaction, so two racing requests
	// cannot both do it.
	type job struct {
		id, createdBy int64
		due, anchor   string
		days          int
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id, created_by, COALESCE(due_date, ''), COALESCE(repeat_anchor, ''), COALESCE(notice_days, ?)
		FROM tasks
		WHERE repeat = ? AND removed_at IS NULL AND stage = 'done' AND done_at IS NOT NULL AND done_at < ?
		ORDER BY id`, defaultNoticeWeeks*7, repeatYearly, cutoff)
	if err != nil {
		return 0, err
	}
	var away []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.createdBy, &j.due, &j.anchor, &j.days); err != nil {
			rows.Close()
			return 0, err
		}
		away = append(away, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, j := range away {
		base := j.due
		if base == "" {
			base = day
		}
		anchor := j.anchor
		if anchor == "" {
			anchor = base[5:]
		}
		newDue := nextYearlyDue(base, anchor)
		shows, err := showsFromFor(newDue, j.days)
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE tasks SET stage = ?, done_at = NULL, due_date = ?, shows_from = ?, repeat_reset_on = NULL, updated_by = created_by, updated_at = ? WHERE id = ?`,
			StageTodo, newDue, shows, nowStr, j.id); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE task_checklist_items SET done_by = NULL, done_at = NULL WHERE task_id = ? AND removed_at IS NULL AND done_at IS NOT NULL`, j.id); err != nil {
			return 0, err
		}
		if err := recordActivity(ctx, tx, j.id, j.createdBy, "yearly_wait", yearlyDetail{By: "comphq", Until: shows}, nowStr); err != nil {
			return 0, err
		}
		changed[j.id] = true
	}
	if len(away) > 0 {
		doneOrder, err := stageOrder(ctx, tx, StageDone, 0, today)
		if err != nil {
			return 0, err
		}
		if err := renumber(ctx, tx, doneOrder); err != nil {
			return 0, err
		}
	}

	// 2. Show.
	rows, err = tx.QueryContext(ctx, `
		SELECT id, created_by, COALESCE(due_date, ''), shows_from
		FROM tasks
		WHERE repeat = ? AND removed_at IS NULL AND stage != 'done' AND shows_from <= ?
		  AND (repeat_reset_on IS NULL OR repeat_reset_on != shows_from)
		ORDER BY shows_from, id`, repeatYearly, day)
	if err != nil {
		return 0, err
	}
	type showing struct {
		id, createdBy int64
		due, shows    string
	}
	var shown []showing
	for rows.Next() {
		var j showing
		if err := rows.Scan(&j.id, &j.createdBy, &j.due, &j.shows); err != nil {
			rows.Close()
			return 0, err
		}
		shown = append(shown, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, j := range shown {
		others, err := stageOrder(ctx, tx, StageTodo, j.id, today)
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE tasks SET stage = ?, position = ?, repeat_reset_on = ?, updated_by = created_by, updated_at = ? WHERE id = ?`,
			StageTodo, len(others)+1, j.shows, nowStr, j.id); err != nil {
			return 0, err
		}
		if err := recordActivity(ctx, tx, j.id, j.createdBy, "yearly_shown", yearlyDetail{By: "comphq", Due: j.due}, nowStr); err != nil {
			return 0, err
		}
		changed[j.id] = true
	}

	if len(changed) > 0 {
		if err := bumpTasksVersion(ctx, tx); err != nil {
			return 0, err
		}
	}
	return len(changed), tx.Commit()
}

// yearlyIDs says which of the given jobs repeat every year, from one query for a
// whole list (as weeklyIDs does for the WEEKLY stamp).
func (s *Store) yearlyIDs(ctx context.Context, taskIDs []int64) (map[int64]bool, error) {
	out := make(map[int64]bool)
	if len(taskIDs) == 0 {
		return out, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(taskIDs)), ",")
	args := make([]any, 0, len(taskIDs)+1)
	args = append(args, repeatYearly)
	for _, id := range taskIDs {
		args = append(args, id)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM tasks WHERE repeat = ? AND id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
