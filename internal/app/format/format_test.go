package format

import (
	"testing"
	"time"
)

func TestDate(t *testing.T) {
	// 19 September 2026 is a Saturday.
	got := Date(time.Date(2026, time.September, 19, 0, 0, 0, 0, time.UTC))
	if want := "Sat 19 Sep 2026"; got != want {
		t.Errorf("Date() = %q, want %q", got, want)
	}
}

func TestClock(t *testing.T) {
	got := Clock(time.Date(2026, time.September, 19, 14, 30, 0, 0, time.UTC))
	if want := "14:30"; got != want {
		t.Errorf("Clock() = %q, want %q", got, want)
	}
}

func TestDateTime(t *testing.T) {
	got := DateTime(time.Date(2026, time.September, 19, 14, 30, 0, 0, time.UTC))
	if want := "Sat 19 Sep 2026, 14:30"; got != want {
		t.Errorf("DateTime() = %q, want %q", got, want)
	}
}

// SPEC gate 4.05: the date field accepts UK order (day first), and still
// accepts the ISO form.
func TestParseDayFirst(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"25/09/2026", "2026-09-25", true},
		{"5/9/2026", "2026-09-05", true},
		{" 05-09-2026 ", "2026-09-05", true},
		{"05.09.2026", "2026-09-05", true},
		{"2026-09-25", "2026-09-25", true},
		{"09/25/2026", "", false}, // month 25 doesn't exist: not silently read the American way
		{"31/02/2026", "", false},
		{"29/02/2028", "2028-02-29", true},
		{"29/02/2026", "", false},
		{"", "", false},
		{"tomorrow", "", false},
		{"25/09/26", "", false},
	}
	for _, c := range cases {
		got, ok := ParseDayFirst(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseDayFirst(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestNormaliseDateKeepsWhatItCannotRead(t *testing.T) {
	today := time.Date(2026, time.September, 22, 0, 0, 0, 0, time.UTC)
	if got := NormaliseDate("25/09/2026", today); got != "2026-09-25" {
		t.Errorf("NormaliseDate(day first) = %q, want 2026-09-25", got)
	}
	if got := NormaliseDate("  next week ", today); got != "next week" {
		t.Errorf("NormaliseDate(unreadable) = %q, want the text kept so validation still refuses it", got)
	}
	if got := NormaliseDate("", today); got != "" {
		t.Errorf("NormaliseDate(empty) = %q, want empty", got)
	}
	// SPEC gate 6.18: NormaliseDate applies the same nearest-year rule as
	// ParseDayFirstNear, not just ParseDayFirst.
	if got := NormaliseDate("25/9", today); got != "2026-09-25" {
		t.Errorf("NormaliseDate(yearless) = %q, want 2026-09-25", got)
	}
}

func TestDayFirstRendersIsoAndLeavesOtherTextAlone(t *testing.T) {
	if got := DayFirst("2026-09-05"); got != "05/09/2026" {
		t.Errorf("DayFirst(iso) = %q, want 05/09/2026", got)
	}
	for _, in := range []string{"", "next week", "31/02/2026"} {
		if got := DayFirst(in); got != in {
			t.Errorf("DayFirst(%q) = %q, want it unchanged", in, got)
		}
	}
	today := time.Date(2026, time.September, 22, 0, 0, 0, 0, time.UTC)
	if got := DayFirst(NormaliseDate("5/9/2026", today)); got != "05/09/2026" {
		t.Errorf("round trip = %q, want 05/09/2026", got)
	}
}

// SPEC gates 6.18, 6.19 (D-82): a date typed without its year means the
// nearest such date to today, and everything ParseDayFirst already
// refuses stays refused.
func TestParseDayFirstNear(t *testing.T) {
	sep22 := time.Date(2026, time.September, 22, 0, 0, 0, 0, time.UTC)
	dec20 := time.Date(2026, time.December, 20, 0, 0, 0, 0, time.UTC)
	leap := time.Date(2028, time.January, 1, 0, 0, 0, 0, time.UTC) // 2028 is a leap year
	nonLeap := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		in    string
		today time.Time
		want  string
		ok    bool
	}{
		// D-82's own worked examples, and the year-boundary case (§4.2).
		{"25/9 on 22 Sep 2026: this year", "25/9", sep22, "2026-09-25", true},
		{"1/7 on 22 Sep 2026: earlier this year, not next July", "1/7", sep22, "2026-07-01", true},
		{"5/1 on 20 Dec 2026: next January, not the one just gone", "5/1", dec20, "2027-01-05", true},
		{"dot separator without a year", "25.9", sep22, "2026-09-25", true},
		{"dash separator without a year", "25-9", sep22, "2026-09-25", true},

		// 29 February: resolves to a real leap year within reach (last,
		// this or next year) rather than being refused outright...
		{"29/2 with a leap year in reach", "29/2", leap, "2028-02-29", true},
		// ...but none of the three candidate years around 1 June 2026
		// (2025, 2026, 2027) is a leap year, so it's refused, same as
		// any other day that doesn't exist.
		{"29/2 with no leap year in reach is refused", "29/2", nonLeap, "", false},

		// Still refused (gate 6.19, D-61 unchanged): a short date that
		// isn't a real day, and American month-first order.
		{"31/9 is not a real day in any year", "31/9", sep22, "", false},
		{"0/5 is not a real day", "0/5", sep22, "", false},
		{"9/25 reads as month 25, not the American 25 September", "9/25", sep22, "", false},

		// Every v1.2 (full-year) accepted form still works unchanged,
		// since ParseDayFirstNear tries ParseDayFirst first.
		{"full year still accepted", "25/09/2026", sep22, "2026-09-25", true},
		{"full year, American order, still refused", "09/25/2026", sep22, "", false},
		{"empty", "", sep22, "", false},
		{"unreadable text", "next week", sep22, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ParseDayFirstNear(c.in, c.today)
			if got != c.want || ok != c.ok {
				t.Errorf("ParseDayFirstNear(%q, %s) = (%q, %v), want (%q, %v)", c.in, c.today.Format("2006-01-02"), got, ok, c.want, c.ok)
			}
		})
	}
}

func TestInitials(t *testing.T) {
	cases := []struct{ name, want string }{
		{"Sam", "S"},
		{"Sam Jones", "SJ"},
		{"sam", "S"},
		{"  Sam   Jones  ", "SJ"},
		{"Sam Middle Jones", "SJ"},
		{"", ""},
	}
	for _, c := range cases {
		if got := Initials(c.name); got != c.want {
			t.Errorf("Initials(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDateShortDropsTheYearOnlyForThisYear(t *testing.T) {
	today := time.Date(2026, time.September, 19, 0, 0, 0, 0, time.UTC)
	if got, want := DateShort(time.Date(2026, time.September, 23, 0, 0, 0, 0, time.UTC), today), "Wed 23 Sep"; got != want {
		t.Errorf("this year = %q, want %q", got, want)
	}
	if got, want := DateShort(time.Date(2027, time.January, 5, 0, 0, 0, 0, time.UTC), today), "Tue 5 Jan 2027"; got != want {
		t.Errorf("another year = %q, want %q", got, want)
	}
}
