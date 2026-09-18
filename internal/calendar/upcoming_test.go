package calendar

import (
	"context"
	"testing"
)

// SPEC gate 3.07: an event's "show ahead" time decides when it first
// appears in Coming up — start − notice ≤ Saturday, counted from
// Saturday, not from the day the page is opened.
func TestUpcomingWithNoticeAppearsOnceItsNoticeHasStarted(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	if _, err := store.Create(ctx, Input{Title: "Card campaign", StartDate: "2026-10-31", NoticeDays: 42}, sam, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.UpcomingWithNotice(ctx, date(t, "2026-09-25"), date(t, "2026-09-19"))
	if err != nil {
		t.Fatalf("UpcomingWithNotice: %v", err)
	}
	if len(got) != 1 || got[0].Title != "Card campaign" || got[0].StartDate != "2026-10-31" || got[0].NoticeDays != 42 {
		t.Fatalf("for Saturday 19 Sep got %+v, want the card campaign on 2026-10-31 with 42 days' notice", got)
	}

	got, err = store.UpcomingWithNotice(ctx, date(t, "2026-09-18"), date(t, "2026-09-12"))
	if err != nil {
		t.Fatalf("UpcomingWithNotice: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("for Saturday 12 Sep got %+v, want nothing (its notice hasn't started)", got)
	}
}

// Events with no show-ahead time never appear early, and events
// already inside the coming week (start ≤ after) are This week's, not
// Coming up's.
func TestUpcomingWithNoticeIgnoresNoNoticeAndThisWeek(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	for _, in := range []Input{
		{Title: "No notice", StartDate: "2026-09-30"},
		{Title: "Exams", StartDate: "2026-09-21", NoticeDays: 14},
		{Title: "Long trip", StartDate: "2026-09-17", EndDate: "2026-09-28", NoticeDays: 14},
	} {
		if _, err := store.Create(ctx, in, sam, fixedNow); err != nil {
			t.Fatalf("Create %q: %v", in.Title, err)
		}
	}

	got, err := store.UpcomingWithNotice(ctx, date(t, "2026-09-25"), date(t, "2026-09-19"))
	if err != nil {
		t.Fatalf("UpcomingWithNotice: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %+v, want nothing", got)
	}
}

// A repeating series contributes only its next qualifying occurrence,
// however many would fit in its notice window.
func TestUpcomingWithNoticeReturnsOnlyTheNextOccurrenceOfASeries(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	if _, err := store.Create(ctx, Input{
		Title: "Staff meeting", StartDate: "2026-09-04", Recurrence: RecurrenceWeekly, NoticeDays: 30,
	}, sam, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.UpcomingWithNotice(ctx, date(t, "2026-09-25"), date(t, "2026-09-19"))
	if err != nil {
		t.Fatalf("UpcomingWithNotice: %v", err)
	}
	if len(got) != 1 || got[0].StartDate != "2026-10-02" {
		t.Errorf("got %+v, want exactly one occurrence, on Fri 2 Oct", got)
	}
}

// SPEC gate 3.08: a cancelled occurrence never shows, and a moved one
// shows at its new date.
func TestUpcomingWithNoticeRespectsCancelledAndMovedOccurrences(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	cancelled, err := store.Create(ctx, Input{
		Title: "Cancelled fair", StartDate: "2026-10-03", Recurrence: RecurrenceWeekly, NoticeDays: 14,
	}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.SetCancelledException(ctx, cancelled.ID, "2026-10-03", sam, fixedNow); err != nil {
		t.Fatalf("SetCancelledException: %v", err)
	}
	moved, err := store.Create(ctx, Input{Title: "Moved concert", StartDate: "2026-10-31", NoticeDays: 42}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.SetMovedException(ctx, moved.ID, "2026-10-31", "2026-10-24", "", "", "", sam, fixedNow); err != nil {
		t.Fatalf("SetMovedException: %v", err)
	}

	got, err := store.UpcomingWithNotice(ctx, date(t, "2026-09-25"), date(t, "2026-09-19"))
	if err != nil {
		t.Fatalf("UpcomingWithNotice: %v", err)
	}
	titles := map[string]string{}
	for _, o := range got {
		titles[o.Title] = o.StartDate
	}
	// The weekly series' 3 Oct is cancelled, so its next is 10 Oct —
	// within 14 days of Saturday 19 Sep? 10 Oct − 14 = 26 Sep > 19 Sep, no.
	if _, ok := titles["Cancelled fair"]; ok {
		t.Errorf("Cancelled fair appeared: %+v", got)
	}
	if titles["Moved concert"] != "2026-10-24" {
		t.Errorf("Moved concert at %q, want its new date 2026-10-24 (all: %+v)", titles["Moved concert"], got)
	}
}
