package calendar

import (
	"context"
	"reflect"
	"testing"
)

// SPEC gate 2.22: events.csv has plain-English headers, readable dates
// and repeat/show-ahead wording, and includes removed events.
func TestExportRows(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	if _, err := store.Create(ctx, Input{
		Title: "Christmas concert", Notes: "Bring programmes", StartDate: "2026-12-12", EndDate: "2026-12-13",
		StartTime: "18:00", EndTime: "20:00", Recurrence: RecurrenceYearly, UntilDate: "2030-12-31", NoticeDays: 42,
	}, sam, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	gone, err := store.Create(ctx, Input{Title: "Cancelled fair", StartDate: "2026-10-01", NoticeDays: 3}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Remove(ctx, gone.ID, sam, fixedNow); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := store.ExportRows(ctx)
	if err != nil {
		t.Fatalf("ExportRows: %v", err)
	}
	want := [][]string{
		{"Title", "Date", "End date", "Start time", "End time", "Repeats", "Until", "Show ahead", "Notes", "Removed"},
		{"Christmas concert", "Sat 12 Dec 2026", "Sun 13 Dec 2026", "18:00", "20:00", "Yearly", "Tue 31 Dec 2030", "6 weeks", "Bring programmes", ""},
		{"Cancelled fair", "Thu 1 Oct 2026", "", "", "", "Never", "", "3 days", "", "Thu 17 Sep 2026, 12:00"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExportRows =\n%v\nwant\n%v", got, want)
	}
}

func TestShowAheadLabel(t *testing.T) {
	for days, want := range map[int]string{0: "Not early", 1: "1 day", 3: "3 days", 7: "1 week", 14: "2 weeks", 10: "10 days"} {
		if got := showAheadLabel(days); got != want {
			t.Errorf("showAheadLabel(%d) = %q, want %q", days, got, want)
		}
	}
}
