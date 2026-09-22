package calendar

import (
	"testing"
	"time"
)

// SPEC gate 6.20 (D-83): the month grid runs Sunday to Saturday. Three
// boundary cases from PLAN-v1.3.md P6-04: a month starting mid-week, one
// starting on a Saturday (a full six-row grid), and one starting on a
// Sunday itself (an exact four-row grid, no spill at the start).
func TestSundayFirstGridBoundaries(t *testing.T) {
	cases := []struct {
		name          string
		year          int
		month         time.Month
		wantFirst     string
		wantLast      string
		wantWeekCount int
	}{
		// September 2026 starts on a Tuesday.
		{"September 2026 (starts Tuesday)", 2026, time.September, "2026-08-30", "2026-10-03", 5},
		// August 2026 starts on a Saturday: the grid runs a full six weeks.
		{"August 2026 (starts Saturday)", 2026, time.August, "2026-07-26", "2026-09-05", 6},
		// February 2026 starts on a Sunday: no spill at the start, exactly
		// four weeks (28 days, no 29th).
		{"February 2026 (starts Sunday)", 2026, time.February, "2026-02-01", "2026-02-28", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			firstOfMonth := time.Date(c.year, c.month, 1, 0, 0, 0, 0, time.UTC)
			lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
			gridStart := sundayOnOrBefore(firstOfMonth)
			gridEnd := saturdayOnOrAfter(lastOfMonth)

			if got := gridStart.Format("2006-01-02"); got != c.wantFirst {
				t.Errorf("gridStart = %s, want %s", got, c.wantFirst)
			}
			if got := gridEnd.Format("2006-01-02"); got != c.wantLast {
				t.Errorf("gridEnd = %s, want %s", got, c.wantLast)
			}
			if gridStart.Weekday() != time.Sunday {
				t.Errorf("gridStart weekday = %s, want Sunday", gridStart.Weekday())
			}
			if gridEnd.Weekday() != time.Saturday {
				t.Errorf("gridEnd weekday = %s, want Saturday", gridEnd.Weekday())
			}

			days := int(gridEnd.Sub(gridStart).Hours()/24) + 1
			if days%7 != 0 {
				t.Fatalf("grid span %d days is not a whole number of weeks", days)
			}
			if weeks := days / 7; weeks != c.wantWeekCount {
				t.Errorf("week count = %d, want %d", weeks, c.wantWeekCount)
			}
		})
	}
}
