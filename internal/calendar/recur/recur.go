// Package recur expands a repeating calendar event into its individual
// occurrences within a date range (SPEC B4's Recurrence section): a
// pure function with no DB access, shared by the Calendar and the
// Briefing.
package recur

import (
	"sort"
	"time"
)

// Recurrence values (SPEC B3's events.recurrence column).
const (
	None    = "none"
	Weekly  = "weekly"
	Monthly = "monthly"
	Yearly  = "yearly"
)

// Exception kinds (SPEC B3's event_exceptions.kind column).
const (
	Moved     = "moved"
	Cancelled = "cancelled"
)

const dateLayout = "2006-01-02"

// Event is the series row occurrences are expanded from (SPEC B3's
// events table).
type Event struct {
	Title      string
	Notes      string
	StartDate  string // "YYYY-MM-DD"
	EndDate    string // "YYYY-MM-DD"; "" means the same day as StartDate
	StartTime  string // "HH:MM"; "" means all day
	EndTime    string // "HH:MM"; "" means none
	Recurrence string // None | Weekly | Monthly | Yearly
	UntilDate  string // "YYYY-MM-DD"; "" means no end
}

// Exception is one event_exceptions row, keyed by the original
// occurrence date it overrides (SPEC B4: "Exceptions are keyed by the
// original date").
type Exception struct {
	OriginalDate string // "YYYY-MM-DD"
	Kind         string // Moved | Cancelled
	NewStartDate string
	NewEndDate   string
	NewStartTime string
	NewEndTime   string
}

// Occurrence is one concrete instance of an event on the calendar.
type Occurrence struct {
	StartDate string
	EndDate   string
	StartTime string
	EndTime   string
	Title     string
	Notes     string
	// OriginalDate is the date this occurrence would have fallen on
	// before any exception — the key a "just this one" edit needs to
	// address it (SPEC B4: "exceptions are keyed by the original
	// date"). Equal to StartDate unless this occurrence has been moved.
	OriginalDate string
}

// Occurrences expands event into every occurrence overlapping [from,
// to] (SPEC B4's Briefing: "occurrences ... overlapping [today, F]"),
// applying exceptions along the way, sorted by start. Title and notes
// always come from the series (SPEC B4) — even a moved occurrence
// only ever has its own dates and times, never its own text, since
// "Change just this one" only offers date/time changes and cancel.
func Occurrences(event Event, exceptions []Exception, from, to time.Time) []Occurrence {
	byOriginal := make(map[string]Exception, len(exceptions))
	for _, ex := range exceptions {
		byOriginal[ex.OriginalDate] = ex
	}

	length := LengthDays(event.StartDate, event.EndDate)
	searchFrom := from.AddDate(0, 0, -length)

	var out []Occurrence
	for _, start := range candidateDates(event, searchFrom, to) {
		startStr := start.Format(dateLayout)
		if _, ok := byOriginal[startStr]; ok {
			// Cancelled never shows; moved shows only at its new date,
			// handled in the loop below — either way, nothing to emit
			// here at the original date.
			continue
		}
		end := start.AddDate(0, 0, length)
		if overlaps(start, end, from, to) {
			out = append(out, Occurrence{
				StartDate:    startStr,
				EndDate:      end.Format(dateLayout),
				StartTime:    event.StartTime,
				EndTime:      event.EndTime,
				Title:        event.Title,
				Notes:        event.Notes,
				OriginalDate: startStr,
			})
		}
	}

	for _, ex := range exceptions {
		if ex.Kind != Moved {
			continue
		}
		start, err := time.Parse(dateLayout, ex.NewStartDate)
		if err != nil {
			continue
		}
		end := start
		if ex.NewEndDate != "" {
			if parsed, err := time.Parse(dateLayout, ex.NewEndDate); err == nil {
				end = parsed
			}
		}
		if overlaps(start, end, from, to) {
			out = append(out, Occurrence{
				StartDate:    ex.NewStartDate,
				EndDate:      end.Format(dateLayout),
				StartTime:    ex.NewStartTime,
				EndTime:      ex.NewEndTime,
				Title:        event.Title,
				Notes:        event.Notes,
				OriginalDate: ex.OriginalDate,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].StartDate != out[j].StartDate {
			return out[i].StartDate < out[j].StartDate
		}
		return out[i].StartTime < out[j].StartTime
	})
	return out
}

// LengthDays is a multi-day event's length in days (SPEC B4: "Multi-day
// events keep their length (end_date − start_date)"). An unset or
// invalid end date means a single-day event. Exported so callers
// outside this package (P2-15's single-occurrence editing) can compute
// the same length without reimplementing it.
func LengthDays(startDate, endDate string) int {
	if endDate == "" {
		return 0
	}
	start, err1 := time.Parse(dateLayout, startDate)
	end, err2 := time.Parse(dateLayout, endDate)
	if err1 != nil || err2 != nil {
		return 0
	}
	days := int(end.Sub(start).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// overlaps reports whether [start, end] intersects [from, to]
// (inclusive on every bound).
func overlaps(start, end, from, to time.Time) bool {
	return !start.After(to) && !end.Before(from)
}

// candidateDates returns every original occurrence start date for
// event's own recurrence rule that could possibly matter for [from,
// to]: not before searchFrom (from, widened by the event's own
// multi-day length, so a multi-day occurrence starting just before
// searchFrom but overlapping into it isn't missed), not after to, and
// not after until_date (SPEC B4: "until_date is inclusive on the
// occurrence start").
func candidateDates(event Event, searchFrom, to time.Time) []time.Time {
	start, err := time.Parse(dateLayout, event.StartDate)
	if err != nil {
		return nil
	}

	hasUntil := false
	var until time.Time
	if event.UntilDate != "" {
		if u, err := time.Parse(dateLayout, event.UntilDate); err == nil {
			hasUntil = true
			until = u
		}
	}
	pastEnd := func(d time.Time) bool {
		return d.After(to) || (hasUntil && d.After(until))
	}

	var out []time.Time
	switch event.Recurrence {
	case Weekly:
		for d := start; !pastEnd(d); d = d.AddDate(0, 0, 7) {
			if !d.Before(searchFrom) {
				out = append(out, d)
			}
		}
	case Monthly:
		// Same day of month, clamped to the month's last day when it
		// doesn't have that many (SPEC B4) — recalculated fresh every
		// month, not carried forward, so a later longer month is back
		// to the exact original day (31 Jan -> 28 Feb -> 31 Mar).
		day := start.Day()
		for cursor := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC); ; cursor = cursor.AddDate(0, 1, 0) {
			d := clampedMonthDate(cursor.Year(), cursor.Month(), day)
			if pastEnd(d) {
				break
			}
			if !d.Before(searchFrom) {
				out = append(out, d)
			}
		}
	case Yearly:
		// Same month and day, with 29 Feb becoming 28 Feb in a
		// non-leap year (SPEC B4) — the only date that can be invalid
		// from one year to the next.
		month := start.Month()
		day := start.Day()
		for year := start.Year(); ; year++ {
			d := clampedYearlyDate(year, month, day)
			if pastEnd(d) {
				break
			}
			if !d.Before(searchFrom) {
				out = append(out, d)
			}
		}
	default: // None: a single, non-repeating occurrence.
		if !pastEnd(start) && !start.Before(searchFrom) {
			out = append(out, start)
		}
	}
	return out
}

func clampedMonthDate(year int, month time.Month, day int) time.Time {
	if last := lastDayOfMonth(year, month); day > last {
		day = last
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func lastDayOfMonth(year int, month time.Month) int {
	firstOfNext := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	return firstOfNext.AddDate(0, 0, -1).Day()
}

func clampedYearlyDate(year int, month time.Month, day int) time.Time {
	if month == time.February && day == 29 && !isLeapYear(year) {
		day = 28
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
