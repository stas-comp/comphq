package holidays

import (
	"fmt"
	"testing"
	"time"
)

// SPEC gates 8.60, 8.62, 8.66, 8.67 (B14.5, D-109). The table below is every US
// federal holiday from 2026 to 2030 and its observed days. The observed weekdays
// are OPM's own published lists (opm.gov, "Federal Holidays", read 21 Nov 2026);
// OPM prints only the weekday a weekend holiday moves to, so the weekend dates
// themselves come from the fixed rules. How the post office treats each weekend
// day is SPEC B14.5's "Postal practice": a Saturday holiday is a no-mail Saturday
// with mail as normal on the Friday before; a Sunday holiday means no mail on the
// Monday after.
//
// Each line is "date kind name".
var table = map[int][]string{
	2026: {
		"2026-01-01 closed New Year's Day",
		"2026-01-19 closed Martin Luther King Jr. Day",
		"2026-02-16 closed Presidents' Day",
		"2026-05-25 closed Memorial Day",
		"2026-06-19 closed Juneteenth",
		"2026-07-03 observed_open Independence Day (observed)",
		"2026-07-04 closed Independence Day",
		"2026-09-07 closed Labor Day",
		"2026-10-12 closed Columbus Day",
		"2026-11-11 closed Veterans Day",
		"2026-11-26 closed Thanksgiving Day",
		"2026-12-25 closed Christmas Day",
	},
	2027: {
		"2027-01-01 closed New Year's Day",
		"2027-01-18 closed Martin Luther King Jr. Day",
		"2027-02-15 closed Presidents' Day",
		"2027-05-31 closed Memorial Day",
		"2027-06-18 observed_open Juneteenth (observed)",
		"2027-06-19 closed Juneteenth",
		"2027-07-04 sunday Independence Day",
		"2027-07-05 closed Independence Day (observed)",
		"2027-09-06 closed Labor Day",
		"2027-10-11 closed Columbus Day",
		"2027-11-11 closed Veterans Day",
		"2027-11-25 closed Thanksgiving Day",
		"2027-12-24 observed_open Christmas Day (observed)",
		"2027-12-25 closed Christmas Day",
		"2027-12-31 observed_open New Year's Day (observed)", // for 1 Jan 2028
	},
	2028: {
		"2028-01-01 closed New Year's Day",
		"2028-01-17 closed Martin Luther King Jr. Day",
		"2028-02-21 closed Presidents' Day",
		"2028-05-29 closed Memorial Day",
		"2028-06-19 closed Juneteenth",
		"2028-07-04 closed Independence Day",
		"2028-09-04 closed Labor Day",
		"2028-10-09 closed Columbus Day",
		"2028-11-10 observed_open Veterans Day (observed)",
		"2028-11-11 closed Veterans Day",
		"2028-11-23 closed Thanksgiving Day",
		"2028-12-25 closed Christmas Day",
	},
	2029: {
		"2029-01-01 closed New Year's Day",
		"2029-01-15 closed Martin Luther King Jr. Day",
		"2029-02-19 closed Presidents' Day",
		"2029-05-28 closed Memorial Day",
		"2029-06-19 closed Juneteenth",
		"2029-07-04 closed Independence Day",
		"2029-09-03 closed Labor Day",
		"2029-10-08 closed Columbus Day",
		"2029-11-11 sunday Veterans Day",
		"2029-11-12 closed Veterans Day (observed)",
		"2029-11-22 closed Thanksgiving Day",
		"2029-12-25 closed Christmas Day",
	},
	2030: {
		"2030-01-01 closed New Year's Day",
		"2030-01-21 closed Martin Luther King Jr. Day",
		"2030-02-18 closed Presidents' Day",
		"2030-05-27 closed Memorial Day",
		"2030-06-19 closed Juneteenth",
		"2030-07-04 closed Independence Day",
		"2030-09-02 closed Labor Day",
		"2030-10-14 closed Columbus Day",
		"2030-11-11 closed Veterans Day",
		"2030-11-28 closed Thanksgiving Day",
		"2030-12-25 closed Christmas Day",
	},
}

func render(days []Day) []string {
	out := make([]string, len(days))
	for i, d := range days {
		out[i] = fmt.Sprintf("%s %s %s", d.Date.Format("2006-01-02"), d.Kind, d.Name)
	}
	return out
}

func date(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		panic(err)
	}
	return t
}

// Gate 8.66: every holiday and observed day 2026-2030 matches the official table, exactly.
func TestGate866FixedTable2026To2030(t *testing.T) {
	for year := 2026; year <= 2030; year++ {
		got := render(Days(date(fmt.Sprintf("%d-01-01", year)), date(fmt.Sprintf("%d-12-31", year))))
		want := table[year]
		if len(got) != len(want) {
			t.Errorf("%d: %d days, want %d\n got  %v\n want %v", year, len(got), len(want), got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%d line %d = %q, want %q", year, i, got[i], want[i])
			}
		}
	}
}

// Gate 8.60: spot checks from the spec's own list.
func TestGate860SpotChecks(t *testing.T) {
	for _, c := range []struct{ date, name string }{
		{"2026-11-26", "Thanksgiving Day"},
		{"2027-01-18", "Martin Luther King Jr. Day"},
		{"2027-05-31", "Memorial Day"},
		{"2028-09-04", "Labor Day"},
		{"2029-10-08", "Columbus Day"},
		{"2030-11-11", "Veterans Day"},
	} {
		days := Days(date(c.date), date(c.date))
		if len(days) != 1 || days[0].Name != c.name || days[0].Kind != Closed {
			t.Errorf("%s = %+v, want %s (closed)", c.date, days, c.name)
		}
	}
	for _, c := range []struct {
		date, name string
		kind       Kind
	}{
		{"2027-06-19", "Juneteenth", Closed},
		{"2027-06-18", "Juneteenth (observed)", ObservedOpen},
		{"2027-07-04", "Independence Day", Sunday},
		{"2027-07-05", "Independence Day (observed)", Closed},
		{"2027-12-25", "Christmas Day", Closed},
		{"2027-12-24", "Christmas Day (observed)", ObservedOpen},
		{"2028-01-01", "New Year's Day", Closed},
		{"2027-12-31", "New Year's Day (observed)", ObservedOpen},
	} {
		days := Days(date(c.date), date(c.date))
		if len(days) != 1 || days[0].Name != c.name || days[0].Kind != c.kind {
			t.Errorf("%s = %+v, want %s (%s)", c.date, days, c.name, c.kind)
		}
	}
}

// Gate 8.60: there are 11 federal holidays in a year (Juneteenth from 2021), with no
// observed day left over from an earlier year in the wrong place.
func TestElevenHolidaysAYearAndJuneteenthOnlyFrom2021(t *testing.T) {
	names := map[string]bool{}
	for _, d := range Days(date("2026-01-01"), date("2026-12-31")) {
		names[d.Name] = true
	}
	for _, n := range []string{"New Year's Day", "Martin Luther King Jr. Day", "Presidents' Day", "Memorial Day", "Juneteenth", "Independence Day",
		"Labor Day", "Columbus Day", "Veterans Day", "Thanksgiving Day", "Christmas Day"} {
		if !names[n] {
			t.Errorf("2026 has no %s", n)
		}
	}
	for year := 1990; year <= 2020; year++ {
		for _, d := range Days(date(fmt.Sprintf("%d-01-01", year)), date(fmt.Sprintf("%d-12-31", year))) {
			if d.Name == "Juneteenth" || d.Name == "Juneteenth (observed)" {
				t.Errorf("%d has %s before it became a federal holiday in 2021", year, d.Name)
			}
		}
	}
	if d := Days(date("2021-06-18"), date("2021-06-19")); len(d) != 2 || d[0].Kind != ObservedOpen || d[1].Kind != Closed {
		t.Errorf("Juneteenth 2021 (a Saturday) = %+v", d)
	}
	if d := Days(date("2020-06-19"), date("2020-06-19")); len(d) != 0 {
		t.Errorf("Juneteenth 2020 = %+v", d)
	}
}

// Gate 8.62, B14.5: the Friday before a Saturday holiday can fall in the year before.
func TestDaysAcrossTheYearBoundaryFindsThe31stOfDecember2027(t *testing.T) {
	days := Days(date("2027-12-28"), date("2028-01-03"))
	got := render(days)
	want := []string{"2027-12-31 observed_open New Year's Day (observed)", "2028-01-01 closed New Year's Day"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Days = %v, want %v", got, want)
	}
	// A range that starts on the 31st still finds it, and one that ends on the 30th does not.
	if d := Days(date("2027-12-31"), date("2027-12-31")); len(d) != 1 || d[0].Kind != ObservedOpen {
		t.Errorf("31 Dec 2027 = %+v", d)
	}
	if d := Days(date("2027-12-26"), date("2027-12-30")); len(d) != 0 {
		t.Errorf("26-30 Dec 2027 = %+v", d)
	}
}

// Ranges are by calendar date: the time of day and the zone of from and to don't matter.
func TestRangesAreInclusiveByCalendarDate(t *testing.T) {
	loc := time.FixedZone("test", -8*3600)
	from := time.Date(2026, 11, 26, 23, 59, 0, 0, loc)
	to := time.Date(2026, 11, 26, 0, 0, 0, 0, loc)
	if d := Days(from, to); len(d) != 1 || d[0].Name != "Thanksgiving Day" {
		t.Errorf("one day = %+v", d)
	}
	if d := Days(date("2026-11-27"), date("2026-11-26")); len(d) != 0 {
		t.Errorf("from after to = %+v", d)
	}
	for _, d := range Days(date("2026-01-01"), date("2030-12-31")) {
		if d.Date.Location() != time.UTC {
			t.Errorf("%s is in %v", d.Date, d.Date.Location())
		}
	}
	// In order.
	all := Days(date("2026-01-01"), date("2030-12-31"))
	for i := 1; i < len(all); i++ {
		if all[i].Date.Before(all[i-1].Date) {
			t.Errorf("out of order: %v then %v", all[i-1].Date, all[i].Date)
		}
	}
}

func TestDescribeIsTheSentenceTheCalendarAndBriefingUse(t *testing.T) {
	for _, c := range []struct {
		day  Day
		want string
	}{
		{Day{Name: "Thanksgiving Day", Kind: Closed}, "Thanksgiving Day · No mail"},
		{Day{Name: "Independence Day", Kind: Sunday}, "Independence Day · No mail"},
		{Day{Name: "Christmas Day (observed)", Kind: ObservedOpen}, "Christmas Day (observed) · Mail as normal"},
	} {
		if got := c.day.Label(); got != c.want {
			t.Errorf("Label = %q, want %q", got, c.want)
		}
	}
	if got := (Day{Kind: Closed}).Explanation(); got != "Federal holiday. Post offices are closed and there is no mail delivery." {
		t.Errorf("closed: %q", got)
	}
	if got := (Day{Kind: Sunday}).Explanation(); got != "Federal holiday. Post offices are closed and there is no mail delivery." {
		t.Errorf("sunday: %q", got)
	}
	if got := (Day{Kind: ObservedOpen}).Explanation(); got != "Federal offices are closed today, but post offices are open and mail is delivered as normal." {
		t.Errorf("observed open: %q", got)
	}
}
