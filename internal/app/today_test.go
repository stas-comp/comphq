package app

import (
	"testing"
	"time"
)

func TestTodayIgnoresOverrideOutsideTestMode(t *testing.T) {
	t.Setenv("COMPHQ_TEST_TODAY", "2026-09-19")

	got := Today(false)
	now := time.Now()
	if got.Year() != now.Year() || got.Month() != now.Month() || got.Day() != now.Day() {
		t.Errorf("Today(false) = %v, want the real current date despite COMPHQ_TEST_TODAY", got)
	}
}

func TestTodayHonoursOverrideInTestMode(t *testing.T) {
	t.Setenv("COMPHQ_TEST_TODAY", "2026-09-19")

	got := Today(true)
	want := time.Date(2026, 9, 19, 0, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("Today(true) = %v, want %v", got, want)
	}
}

func TestTodayFallsBackToRealDateWhenOverrideUnset(t *testing.T) {
	t.Setenv("COMPHQ_TEST_TODAY", "")

	got := Today(true)
	now := time.Now()
	if got.Year() != now.Year() || got.Month() != now.Month() || got.Day() != now.Day() {
		t.Errorf("Today(true) with no override = %v, want the real current date", got)
	}
}
