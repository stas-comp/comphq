package main

import (
	"context"
	"log"
	"time"

	"github.com/stas-comp/comphq/internal/briefing"
	"github.com/stas-comp/comphq/internal/calendar"
	"github.com/stas-comp/comphq/internal/questions"
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

// resetWeekly runs the weekly and the yearly pass (SPEC B13.5, B14.4): the
// Briefing is one of the places that shows jobs, and they must have caught up
// before it does.
func (a briefingTasks) resetWeekly(ctx context.Context) {
	if a.today == nil {
		return
	}
	if _, err := a.store.ResetWeekly(ctx, a.today()); err != nil {
		log.Printf("briefing: weekly reset: %v", err)
	}
	if _, err := a.store.ResetYearly(ctx, a.today()); err != nil {
		log.Printf("briefing: yearly pass: %v", err)
	}
}

func (a briefingTasks) Tasks(ctx context.Context, dueOnOrBefore time.Time) ([]briefing.Task, error) {
	a.resetWeekly(ctx)
	today := dueOnOrBefore
	if a.today != nil {
		today = a.today()
	}
	rows, err := a.store.BriefingTasks(ctx, dueOnOrBefore, today)
	if err != nil {
		return nil, err
	}
	out := make([]briefing.Task, len(rows))
	for i, r := range rows {
		people := make([]briefing.Person, len(r.Assignees))
		for j, p := range r.Assignees {
			people[j] = briefing.Person{ID: p.PersonID, Name: p.Name, Removed: p.Removed}
		}
		out[i] = briefing.Task{ID: r.ID, Title: r.Title, Notes: r.Notes, DueDate: r.DueDate, Position: r.Position, People: people, Weekly: r.Weekly, Yearly: r.Yearly}
	}
	return out, nil
}

func (a briefingTasks) Version(ctx context.Context) (int64, error) {
	a.resetWeekly(ctx)
	return a.store.Version(ctx)
}

// briefingHolidays adapts the Calendar's holidays to briefing.HolidaySource (SPEC
// B14.5): the one place that knows both shapes.
type briefingHolidays struct{}

func (briefingHolidays) HolidaysBetween(from, to time.Time) []briefing.Holiday {
	days := calendar.HolidaysBetween(from, to)
	out := make([]briefing.Holiday, len(days))
	for i, d := range days {
		out[i] = briefing.Holiday{
			Date: d.Date.Format("2006-01-02"), Name: d.Name, Label: d.Label(), Explanation: d.Explanation(), Kind: string(d.Kind),
		}
	}
	return out
}

// briefingQuestions adapts *questions.Store to briefing.QuestionSource (SPEC B14.6).
type briefingQuestions struct {
	store *questions.Store
	today func() time.Time
}

func (a briefingQuestions) Waiting(ctx context.Context, limit int) ([]briefing.WaitingQuestion, int, error) {
	rows, total, err := a.store.WaitingOldest(ctx, limit, a.today())
	if err != nil {
		return nil, 0, err
	}
	out := make([]briefing.WaitingQuestion, len(rows))
	for i, r := range rows {
		out[i] = briefing.WaitingQuestion{ID: r.ID, Text: r.Text, AskedBy: r.AskedBy, AskedLabel: r.AskedLabel, Answers: r.AnswerCount}
	}
	return out, total, nil
}

func (a briefingQuestions) Version(ctx context.Context) (int64, error) {
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
