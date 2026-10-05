package main

import (
	"context"
	"log"
	"time"

	"github.com/stas-comp/comphq/internal/briefing"
	"github.com/stas-comp/comphq/internal/calendar"
	"github.com/stas-comp/comphq/internal/tasks"
)

// briefingTasks adapts *tasks.Store to briefing.TaskSource (SPEC A8),
// converting between each section's own shapes — like taskDeadlines,
// this is the one place allowed to know about both sections (SPEC B2).
type briefingTasks struct {
	store *tasks.Store
	// today is the app's own idea of today (app.Today), so the Saturday reset
	// that runs before the Briefing reads jobs agrees with the rest of the app
	// and with test mode's fixed date.
	today func() time.Time
}

// resetWeekly runs the weekly reset (SPEC B13.5): the Briefing is one of the
// places that shows jobs, and the reset must have caught up before it does.
func (a briefingTasks) resetWeekly(ctx context.Context) {
	if a.today == nil {
		return
	}
	if _, err := a.store.ResetWeekly(ctx, a.today()); err != nil {
		log.Printf("briefing: weekly reset: %v", err)
	}
}

func (a briefingTasks) Tasks(ctx context.Context, dueOnOrBefore time.Time) ([]briefing.Task, error) {
	a.resetWeekly(ctx)
	rows, err := a.store.BriefingTasks(ctx, dueOnOrBefore)
	if err != nil {
		return nil, err
	}
	out := make([]briefing.Task, len(rows))
	for i, r := range rows {
		people := make([]briefing.Person, len(r.Assignees))
		for j, p := range r.Assignees {
			people[j] = briefing.Person{ID: p.PersonID, Name: p.Name, Removed: p.Removed}
		}
		out[i] = briefing.Task{ID: r.ID, Title: r.Title, Notes: r.Notes, DueDate: r.DueDate, Position: r.Position, People: people}
	}
	return out, nil
}

func (a briefingTasks) Version(ctx context.Context) (int64, error) {
	a.resetWeekly(ctx)
	return a.store.Version(ctx)
}

// briefingEvents adapts *calendar.Store to briefing.EventSource.
type briefingEvents struct{ store *calendar.Store }

func (a briefingEvents) Occurrences(ctx context.Context, from, to time.Time) ([]briefing.Occurrence, error) {
	rows, err := a.store.Occurrences(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]briefing.Occurrence, len(rows))
	for i, r := range rows {
		out[i] = briefingOccurrence(r, 0)
	}
	return out, nil
}

func (a briefingEvents) UpcomingWithNotice(ctx context.Context, after, saturday time.Time) ([]briefing.Occurrence, error) {
	rows, err := a.store.UpcomingWithNotice(ctx, after, saturday)
	if err != nil {
		return nil, err
	}
	out := make([]briefing.Occurrence, len(rows))
	for i, r := range rows {
		out[i] = briefingOccurrence(r.CalendarOccurrence, r.NoticeDays)
	}
	return out, nil
}

func (a briefingEvents) Version(ctx context.Context) (int64, error) { return a.store.Version(ctx) }

func briefingOccurrence(o calendar.CalendarOccurrence, noticeDays int) briefing.Occurrence {
	return briefing.Occurrence{
		EventID:      o.EventID,
		Title:        o.Title,
		Notes:        o.Notes,
		StartDate:    o.StartDate,
		EndDate:      o.EndDate,
		StartTime:    o.StartTime,
		EndTime:      o.EndTime,
		OriginalDate: o.OriginalDate,
		NoticeDays:   noticeDays,
	}
}
