// Package calendar holds Calendar events and their recurrence (SPEC A7).
package calendar

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
	"github.com/stas-comp/comphq/internal/calendar/recur"
)

// Recurrence values (SPEC B3's events.recurrence column), re-exported
// from recur so callers never need to import both packages just to
// name one.
const (
	RecurrenceNone    = recur.None
	RecurrenceWeekly  = recur.Weekly
	RecurrenceMonthly = recur.Monthly
	RecurrenceYearly  = recur.Yearly
)

// Recurrences lists every valid value, in the order SPEC gate 2.13's
// repeats field presents them: Never / Weekly / Monthly / Yearly.
var Recurrences = []string{RecurrenceNone, RecurrenceWeekly, RecurrenceMonthly, RecurrenceYearly}

// Event is one events row (SPEC B3), with its last-changed person's
// name resolved and UpdatedAt pre-formatted (SPEC gate 2.20: "Last
// changed by name · date time").
type Event struct {
	ID            int64
	Title         string
	Notes         string
	StartDate     string // "YYYY-MM-DD"
	EndDate       string // "YYYY-MM-DD"; "" means single-day
	StartTime     string // "HH:MM"; "" means all day
	EndTime       string // "HH:MM"; "" means none
	Recurrence    string
	UntilDate     string // "YYYY-MM-DD"; "" means no end
	NoticeDays    int
	UpdatedByName string
	UpdatedAt     string // pre-formatted, SPEC A3: "Sat 19 Sep 2026, 14:30"
}

// CalendarOccurrence is one occurrence ready to render, with the
// series id it belongs to so a chip can link to /calendar/events/{id}.
type CalendarOccurrence struct {
	EventID int64
	recur.Occurrence
}

// Store reads and writes events and their exceptions.
type Store struct {
	DB *sql.DB
}

// ErrEventNotFound is returned by Get and Update for a removed or
// non-existent id.
var ErrEventNotFound = errors.New("that event isn't there any more")

// Validation errors (SPEC gate 2.13's required/consistency rules).
var (
	ErrEmptyTitle            = errors.New("title can't be empty")
	ErrEndDateBeforeStart    = errors.New("end date can't be before the start date")
	ErrEndTimeNeedsStartTime = errors.New("end time needs a start time")
	ErrInvalidRecurrence     = errors.New("that isn't a valid repeat option")
)

// Input is what adding or editing an event takes (SPEC gate 2.13).
type Input struct {
	Title      string
	Notes      string
	StartDate  string
	EndDate    string
	StartTime  string
	EndTime    string
	Recurrence string // "" defaults to RecurrenceNone
	UntilDate  string
	NoticeDays int
}

func validRecurrence(r string) bool {
	for _, v := range Recurrences {
		if v == r {
			return true
		}
	}
	return false
}

// normalize trims the title and defaults recurrence, then runs SPEC
// gate 2.13's validation: title required, end date not before start,
// end time needs a start time.
func normalize(input Input) (Input, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return Input{}, ErrEmptyTitle
	}
	if input.Recurrence == "" {
		input.Recurrence = RecurrenceNone
	}
	if !validRecurrence(input.Recurrence) {
		return Input{}, ErrInvalidRecurrence
	}
	if input.EndDate != "" && input.EndDate < input.StartDate {
		return Input{}, ErrEndDateBeforeStart
	}
	if input.EndTime != "" && input.StartTime == "" {
		return Input{}, ErrEndTimeNeedsStartTime
	}
	return input, nil
}

// Create adds a new event series (SPEC gate 2.13).
func (s *Store) Create(ctx context.Context, input Input, actorID int64, now time.Time) (Event, error) {
	input, err := normalize(input)
	if err != nil {
		return Event{}, err
	}

	nowStr := now.UTC().Format(time.RFC3339)
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO events (title, notes, start_date, end_date, start_time, end_time, recurrence, until_date, notice_days, created_by, created_at, updated_by, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, input.Title, input.Notes, input.StartDate, input.EndDate, input.StartTime, input.EndTime, input.Recurrence, input.UntilDate, input.NoticeDays, actorID, nowStr, actorID, nowStr)
	if err != nil {
		return Event{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Event{}, err
	}
	return s.Get(ctx, id)
}

// Update edits the series row (SPEC B4: "Change all edits the series
// row") — occurrence-level "just this one" editing is P2-15's own job.
func (s *Store) Update(ctx context.Context, id int64, input Input, actorID int64, now time.Time) error {
	input, err := normalize(input)
	if err != nil {
		return err
	}

	nowStr := now.UTC().Format(time.RFC3339)
	res, err := s.DB.ExecContext(ctx, `
		UPDATE events SET title = ?, notes = ?, start_date = ?, end_date = ?, start_time = ?, end_time = ?,
			recurrence = ?, until_date = ?, notice_days = ?, updated_by = ?, updated_at = ?
		WHERE id = ? AND removed_at IS NULL
	`, input.Title, input.Notes, input.StartDate, input.EndDate, input.StartTime, input.EndTime,
		input.Recurrence, input.UntilDate, input.NoticeDays, actorID, nowStr, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrEventNotFound
	}
	return nil
}

// Get returns one event by id, for the details/edit page (SPEC gate 2.20).
func (s *Store) Get(ctx context.Context, id int64) (Event, error) {
	var e Event
	var endDate, startTime, endTime, untilDate sql.NullString
	var updatedAt string

	err := s.DB.QueryRowContext(ctx, `
		SELECT e.id, e.title, e.notes, e.start_date, e.end_date, e.start_time, e.end_time,
			e.recurrence, e.until_date, e.notice_days, e.updated_at, p.name
		FROM events e JOIN people p ON p.id = e.updated_by
		WHERE e.id = ? AND e.removed_at IS NULL
	`, id).Scan(&e.ID, &e.Title, &e.Notes, &e.StartDate, &endDate, &startTime, &endTime,
		&e.Recurrence, &untilDate, &e.NoticeDays, &updatedAt, &e.UpdatedByName)
	if err == sql.ErrNoRows {
		return Event{}, ErrEventNotFound
	}
	if err != nil {
		return Event{}, err
	}

	e.EndDate = endDate.String
	e.StartTime = startTime.String
	e.EndTime = endTime.String
	e.UntilDate = untilDate.String
	if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		e.UpdatedAt = format.DateTime(t)
	} else {
		e.UpdatedAt = updatedAt
	}
	return e, nil
}

// Occurrences expands every non-removed event overlapping [from, to]
// (SPEC gates 2.12/2.14), for the month and list views alike.
func (s *Store) Occurrences(ctx context.Context, from, to time.Time) ([]CalendarOccurrence, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, title, notes, start_date, end_date, start_time, end_time, recurrence, until_date
		FROM events WHERE removed_at IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type series struct {
		id    int64
		event recur.Event
	}
	var all []series
	for rows.Next() {
		var sr series
		var endDate, startTime, endTime, untilDate sql.NullString
		if err := rows.Scan(&sr.id, &sr.event.Title, &sr.event.Notes, &sr.event.StartDate, &endDate,
			&startTime, &endTime, &sr.event.Recurrence, &untilDate); err != nil {
			return nil, err
		}
		sr.event.EndDate = endDate.String
		sr.event.StartTime = startTime.String
		sr.event.EndTime = endTime.String
		sr.event.UntilDate = untilDate.String
		all = append(all, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []CalendarOccurrence
	for _, sr := range all {
		exceptions, err := s.exceptionsFor(ctx, sr.id)
		if err != nil {
			return nil, err
		}
		for _, occ := range recur.Occurrences(sr.event, exceptions, from, to) {
			out = append(out, CalendarOccurrence{EventID: sr.id, Occurrence: occ})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].StartDate != out[j].StartDate {
			return out[i].StartDate < out[j].StartDate
		}
		return out[i].StartTime < out[j].StartTime
	})
	return out, nil
}

// exceptionsFor returns one event's own event_exceptions rows — always
// empty today (nothing writes to this table until P2-15), but
// Occurrences already needs the real query wired to the real table its
// own migration creates.
func (s *Store) exceptionsFor(ctx context.Context, eventID int64) ([]recur.Exception, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT original_date, kind, new_start_date, new_end_date, new_start_time, new_end_time
		FROM event_exceptions WHERE event_id = ?
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []recur.Exception
	for rows.Next() {
		var ex recur.Exception
		var newStartDate, newEndDate, newStartTime, newEndTime sql.NullString
		if err := rows.Scan(&ex.OriginalDate, &ex.Kind, &newStartDate, &newEndDate, &newStartTime, &newEndTime); err != nil {
			return nil, err
		}
		ex.NewStartDate = newStartDate.String
		ex.NewEndDate = newEndDate.String
		ex.NewStartTime = newStartTime.String
		ex.NewEndTime = newEndTime.String
		out = append(out, ex)
	}
	return out, rows.Err()
}
