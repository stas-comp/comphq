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
	if got := NormaliseDate("25/09/2026"); got != "2026-09-25" {
		t.Errorf("NormaliseDate(day first) = %q, want 2026-09-25", got)
	}
	if got := NormaliseDate("  next week "); got != "next week" {
		t.Errorf("NormaliseDate(unreadable) = %q, want the text kept so validation still refuses it", got)
	}
	if got := NormaliseDate(""); got != "" {
		t.Errorf("NormaliseDate(empty) = %q, want empty", got)
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
	if got := DayFirst(NormaliseDate("5/9/2026")); got != "05/09/2026" {
		t.Errorf("round trip = %q, want 05/09/2026", got)
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
