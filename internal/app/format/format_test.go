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
