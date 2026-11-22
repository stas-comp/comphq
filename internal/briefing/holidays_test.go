package briefing

import (
	"strings"
	"testing"
)

// The days the Calendar would hand over (the Briefing never imports it, SPEC B2, so
// the few that matter here are written out by hand; the rules themselves are tested
// in internal/calendar/holidays).
var testHolidays = []Holiday{
	{Date: "2026-11-11", Name: "Veterans Day", Kind: "closed"},
	{Date: "2026-11-26", Name: "Thanksgiving Day", Kind: "closed"},
	{Date: "2026-12-25", Name: "Christmas Day", Kind: "closed"},
	{Date: "2027-07-04", Name: "Independence Day", Kind: "sunday"},
	{Date: "2027-07-05", Name: "Independence Day (observed)", Kind: "closed"},
	{Date: "2027-12-24", Name: "Christmas Day (observed)", Kind: "observed_open"},
	{Date: "2027-12-25", Name: "Christmas Day", Kind: "closed"},
	{Date: "2027-12-31", Name: "New Year's Day (observed)", Kind: "observed_open"},
	{Date: "2028-01-01", Name: "New Year's Day", Kind: "closed"},
}

const (
	noMail  = "Federal holiday. Post offices are closed and there is no mail delivery."
	asUsual = "Federal offices are closed today, but post offices are open and mail is delivered as normal."
)

// holidaysFor is the test holidays from from to to, labelled as the Calendar labels them.
func holidaysFor(t *testing.T, from, to string) []Holiday {
	t.Helper()
	var out []Holiday
	for _, h := range testHolidays {
		if h.Date < from || h.Date > to {
			continue
		}
		h.Label, h.Explanation = h.Name+" · No mail", noMail
		if h.Kind == "observed_open" {
			h.Label, h.Explanation = h.Name+" · Mail as normal", asUsual
		}
		out = append(out, h)
	}
	return out
}

func titles(cards []EventCard) []string {
	var out []string
	for _, c := range cards {
		out = append(out, c.Title+"|"+c.Stamp)
	}
	return out
}

func build(t *testing.T, today string) Briefing {
	t.Helper()
	d := day(t, today)
	return Build(Input{Today: d, Holidays: holidaysFor(t, today, SaturdayFor(d).AddDate(0, 0, 14).Format(dateLayout))})
}

// Gates 8.65, B14.7: Thanksgiving 2026 (Thu 26 Nov) on the five test Saturdays.
func TestGate865ThanksgivingMovesFromNothingToComingUpToThisWeek(t *testing.T) {
	// Sat 7 Nov: Thanksgiving is more than 2 weeks past the Saturday's Friday: not shown anywhere.
	b := build(t, "2026-11-07")
	for _, c := range append(append([]EventCard{}, b.ThisWeek...), b.ComingUp...) {
		if strings.Contains(c.Title, "Thanksgiving") {
			t.Errorf("Thanksgiving is on the page on 7 Nov: %v", c.Title)
		}
	}
	// (Veterans Day, Wed 11 Nov, is in This week that Saturday.)
	if got := titles(b.ThisWeek); len(got) != 1 || got[0] != "Veterans Day · No mail|HOLIDAY" {
		t.Errorf("7 Nov this week = %v", got)
	}
	if len(b.ComingUp) != 0 {
		t.Errorf("7 Nov coming up = %v", titles(b.ComingUp))
	}
}

func TestGate865ThanksgivingFrom14Nov(t *testing.T) {
	b := build(t, "2026-11-14")
	if got := titles(b.ComingUp); len(got) != 1 || got[0] != "Thanksgiving Day · No mail|IN 12 DAYS" {
		t.Errorf("14 Nov coming up = %v", got)
	}
	for _, c := range b.ThisWeek {
		if strings.Contains(c.Title, "Thanksgiving") {
			t.Errorf("Thanksgiving is in This week on 14 Nov")
		}
	}
	for _, c := range b.ComingUp {
		if !c.Holiday || c.Stamp != "IN 12 DAYS" || c.StampKind != "date" || c.Explanation == "" {
			t.Errorf("coming up card = %+v", c)
		}
	}
}

func TestGate865ThanksgivingInThisWeekOn21Nov(t *testing.T) {
	b := build(t, "2026-11-21")
	got := titles(b.ThisWeek)
	if len(got) != 1 || got[0] != "Thanksgiving Day · No mail|HOLIDAY" {
		t.Errorf("21 Nov this week = %v", got)
	}
	if len(b.ComingUp) != 0 {
		t.Errorf("coming up = %v", titles(b.ComingUp))
	}
	if c := b.ThisWeek[0]; !c.Holiday || c.StampKind != "holiday" || c.DateLabel != "Thu 26 Nov 2026" {
		t.Errorf("card = %+v", c)
	}
}

func TestGate865Sunday4Jul2027AndNoMailMondayAreBothInThisWeekOn3Jul(t *testing.T) {
	b := build(t, "2027-07-03")
	got := titles(b.ThisWeek)
	want := []string{"Independence Day · No mail|HOLIDAY", "Independence Day (observed) · No mail|HOLIDAY"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("3 Jul 2027 this week = %v, want %v", got, want)
	}
}

func TestGate865WhenTheSaturdayItselfIsANoMailDay(t *testing.T) {
	b := build(t, "2027-12-25")
	got := titles(b.ThisWeek)
	// Fri 31 Dec is the observed New Year's Day for Sat 1 Jan: in This week, mail as normal.
	want := []string{"Christmas Day · No mail today|HOLIDAY", "New Year's Day (observed) · Mail as normal|HOLIDAY"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("25 Dec 2027 this week = %v, want %v", got, want)
	}
	if !b.IsToday {
		t.Errorf("25 Dec 2027 is the Saturday itself")
	}
}

// Holidays go before an event on the same day, and an event-only week is unchanged.
func TestHolidaysComeBeforeEventsOnTheSameDay(t *testing.T) {
	today := day(t, "2026-11-21")
	b := Build(Input{
		Today:    today,
		Holidays: holidaysFor(t, "2026-11-21", "2026-12-05"),
		ThisWeek: []Occurrence{
			{EventID: 1, Title: "Early event", StartDate: "2026-11-24"},
			{EventID: 2, Title: "Turkey lunch", StartDate: "2026-11-26"},
			{EventID: 3, Title: "Late event", StartDate: "2026-11-27"},
		},
	})
	got := titles(b.ThisWeek)
	want := []string{"Early event|TUE 24 NOV", "Thanksgiving Day · No mail|HOLIDAY", "Turkey lunch|THU 26 NOV", "Late event|FRI 27 NOV"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("card %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// Observed days with mail as normal never go in Coming up, and Just mine keeps every holiday.
func TestObservedOpenDaysAreOnlyInThisWeekAndJustMineKeepsHolidays(t *testing.T) {
	b := build(t, "2027-12-11") // Christmas Eve 2027 (observed, mail as normal) is 13 days ahead
	for _, c := range b.ComingUp {
		if strings.Contains(c.Title, "observed") && strings.Contains(c.Title, "Mail as normal") {
			t.Errorf("a mail-as-normal day in Coming up: %v", c.Title)
		}
	}
	b2 := build(t, "2026-11-21")
	if got := b2.JustMine().ThisWeek; len(got) != len(b2.ThisWeek) {
		t.Errorf("Just mine hid holidays")
	}
}
