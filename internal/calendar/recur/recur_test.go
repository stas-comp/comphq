package recur

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func startDates(occs []Occurrence) []string {
	dates := make([]string, len(occs))
	for i, o := range occs {
		dates[i] = o.StartDate
	}
	return dates
}

func assertDates(t *testing.T, got []Occurrence, want []string) {
	t.Helper()
	gotDates := startDates(got)
	if len(gotDates) != len(want) {
		t.Fatalf("dates = %v, want %v", gotDates, want)
	}
	for i, w := range want {
		if gotDates[i] != w {
			t.Errorf("dates = %v, want %v", gotDates, want)
			return
		}
	}
}

// SPEC B4: "Weekly: every 7 days from start_date."
func TestWeeklyOnTheSameWeekday(t *testing.T) {
	event := Event{Title: "Standup", StartDate: "2026-09-07", Recurrence: Weekly}
	got := Occurrences(event, nil, date("2026-09-01"), date("2026-09-30"))
	assertDates(t, got, []string{"2026-09-07", "2026-09-14", "2026-09-21", "2026-09-28"})
}

// SPEC B4: "Monthly: same day of month, clamped to the month's last
// day" — recalculated fresh every month, not carried forward, so 31
// Jan clamps to 28 Feb 2027 (not a leap year) but is back to a full 31
// Mar.
func TestMonthlyOnThe31stClamped(t *testing.T) {
	event := Event{Title: "Rent", StartDate: "2027-01-31", Recurrence: Monthly}
	got := Occurrences(event, nil, date("2027-01-01"), date("2027-03-31"))
	assertDates(t, got, []string{"2027-01-31", "2027-02-28", "2027-03-31"})
}

// SPEC B4: "Yearly: ... 29 Feb becoming 28 Feb in non-leap years."
func TestYearlyOn29FebClampsInNonLeapYears(t *testing.T) {
	event := Event{Title: "Leap birthday", StartDate: "2024-02-29", Recurrence: Yearly}
	got := Occurrences(event, nil, date("2024-01-01"), date("2028-12-31"))
	assertDates(t, got, []string{"2024-02-29", "2025-02-28", "2026-02-28", "2027-02-28", "2028-02-29"})
}

// SPEC B4: "until_date is inclusive on the occurrence start."
func TestUntilDateBoundaryIsInclusive(t *testing.T) {
	event := Event{Title: "Sprint check-in", StartDate: "2026-01-01", Recurrence: Weekly, UntilDate: "2026-01-15"}
	got := Occurrences(event, nil, date("2026-01-01"), date("2026-02-01"))
	assertDates(t, got, []string{"2026-01-01", "2026-01-08", "2026-01-15"})
}

// SPEC B4: "Multi-day events keep their length (end_date - start_date)."
// An occurrence starting before the query window but overlapping into
// it still appears, with its full original span intact.
func TestMultiDayAcrossMonthEnd(t *testing.T) {
	event := Event{Title: "Offsite", StartDate: "2026-01-30", EndDate: "2026-02-02", Recurrence: None}
	got := Occurrences(event, nil, date("2026-01-01"), date("2026-01-31"))
	if len(got) != 1 {
		t.Fatalf("occurrences = %v, want exactly 1", got)
	}
	if got[0].StartDate != "2026-01-30" || got[0].EndDate != "2026-02-02" {
		t.Errorf("occurrence = %+v, want StartDate=2026-01-30 EndDate=2026-02-02", got[0])
	}
}

// SPEC gate 2.15: a yearly Christmas concert on Sat 12 Dec 2026, moved
// with "Change just this one" to Sat 19 Dec 2026, shows on 19 Dec 2026
// and still shows on 12 Dec 2027 (unaffected — exceptions are keyed by
// the original date only).
func TestMovedExactGate215Dates(t *testing.T) {
	event := Event{Title: "Christmas concert", StartDate: "2026-12-12", Recurrence: Yearly}
	exceptions := []Exception{
		{OriginalDate: "2026-12-12", Kind: Moved, NewStartDate: "2026-12-19"},
	}
	got := Occurrences(event, exceptions, date("2026-01-01"), date("2027-12-31"))
	assertDates(t, got, []string{"2026-12-19", "2027-12-12"})

	// OriginalDate is what a "just this one" edit keys off — the moved
	// occurrence's is its old date, not its new one, and an unaffected
	// occurrence's is simply its own date.
	if got[0].OriginalDate != "2026-12-12" {
		t.Errorf("moved occurrence's OriginalDate = %q, want 2026-12-12", got[0].OriginalDate)
	}
	if got[1].OriginalDate != "2027-12-12" {
		t.Errorf("unaffected occurrence's OriginalDate = %q, want 2027-12-12", got[1].OriginalDate)
	}
}

// SPEC B4: "cancelled removes the occurrence."
func TestCancelledOccurrenceIsRemoved(t *testing.T) {
	event := Event{Title: "Weekly sync", StartDate: "2026-03-02", Recurrence: Weekly}
	exceptions := []Exception{
		{OriginalDate: "2026-03-09", Kind: Cancelled},
	}
	got := Occurrences(event, exceptions, date("2026-03-01"), date("2026-03-31"))
	assertDates(t, got, []string{"2026-03-02", "2026-03-16", "2026-03-23", "2026-03-30"})
}

// A moved exception whose original date falls well outside [from, to]
// still appears, because its new date falls inside it — the query
// window is about the occurrence's effective date, not its original one.
func TestMovedExceptionOutsideWindowAppearsWhenNewDateIsInside(t *testing.T) {
	event := Event{Title: "One-off review", StartDate: "2020-06-01", Recurrence: Yearly}
	exceptions := []Exception{
		{OriginalDate: "2020-06-01", Kind: Moved, NewStartDate: "2026-09-20"},
	}
	got := Occurrences(event, exceptions, date("2026-09-01"), date("2026-09-30"))
	assertDates(t, got, []string{"2026-09-20"})
}

// A query range spanning several years returns one occurrence per year.
func TestRangeAcrossYears(t *testing.T) {
	event := Event{Title: "Annual gala", StartDate: "2025-11-01", Recurrence: Yearly}
	got := Occurrences(event, nil, date("2025-01-01"), date("2027-12-31"))
	assertDates(t, got, []string{"2025-11-01", "2026-11-01", "2027-11-01"})
}

// SPEC B4: "Change all edits the series row ... title and notes always
// come from the series" — even a moved occurrence's title/notes still
// come from the series, never carried on the exception itself.
func TestMovedOccurrenceStillUsesSeriesTitleAndNotes(t *testing.T) {
	event := Event{Title: "Christmas concert", Notes: "Bring programmes", StartDate: "2026-12-12", Recurrence: Yearly}
	exceptions := []Exception{
		{OriginalDate: "2026-12-12", Kind: Moved, NewStartDate: "2026-12-19", NewStartTime: "18:00"},
	}
	got := Occurrences(event, exceptions, date("2026-12-01"), date("2026-12-31"))
	if len(got) != 1 {
		t.Fatalf("occurrences = %v, want exactly 1", got)
	}
	if got[0].Title != "Christmas concert" || got[0].Notes != "Bring programmes" {
		t.Errorf("moved occurrence = %+v, want the series' own title/notes", got[0])
	}
	if got[0].StartTime != "18:00" {
		t.Errorf("moved occurrence StartTime = %q, want 18:00 (from the exception)", got[0].StartTime)
	}
}
