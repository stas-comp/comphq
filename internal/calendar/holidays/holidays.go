// Package holidays works out the US federal holidays for any year (SPEC B14.5,
// D-109, gates 8.60-8.67). It is pure: no database, no clock and no network, so
// there is nothing to store, export or migrate.
//
// The 11 holidays follow fixed rules (a fixed date, or the nth or last weekday of
// a month). The office cares about mail, so a holiday that falls at the weekend
// follows the Postal Service's practice rather than the federal offices':
//
//   - a Saturday holiday is a no-mail Saturday, and the Friday before it is
//     "(observed)" with mail as normal (federal offices are closed that Friday,
//     post offices are open);
//   - a Sunday holiday is shown on the Sunday (there is never mail on Sunday), and
//     the Monday after it is "(observed)" with no mail.
package holidays

import (
	"sort"
	"time"
)

// Kind says what a holiday means for the mail.
type Kind string

const (
	// Closed is the holiday itself on a weekday or a Saturday, or the observed
	// Monday after a Sunday holiday. No mail.
	Closed Kind = "closed"
	// Sunday is the holiday itself on a Sunday: there is never mail on Sunday, so
	// it is shown plainly with "No mail".
	Sunday Kind = "sunday"
	// ObservedOpen is the Friday before a Saturday holiday: federal offices are
	// closed, but the mail is as normal.
	ObservedOpen Kind = "observed_open"
)

// Day is one holiday on one date.
type Day struct {
	Date time.Time
	Name string
	Kind Kind
}

// Label is the words the Calendar chip and the Briefing show, for example
// "Thanksgiving Day · No mail" or "Christmas Day (observed) · Mail as normal".
func (d Day) Label() string {
	if d.Kind == ObservedOpen {
		return d.Name + " · Mail as normal"
	}
	return d.Name + " · No mail"
}

// Explanation is the short sentence the chip carries for hovering, the keyboard
// and a screen reader (gate 8.63).
func (d Day) Explanation() string {
	if d.Kind == ObservedOpen {
		return "Federal offices are closed today, but post offices are open and mail is delivered as normal."
	}
	return "Federal holiday. Post offices are closed and there is no mail delivery."
}

// rule says when a holiday falls in a year: a fixed date, or the nth weekday of a
// month (n = 1..4), or the last weekday of a month (n = -1).
type rule struct {
	name  string
	month time.Month
	day   int          // a fixed day of the month, or 0
	wd    time.Weekday // for the nth or last weekday
	nth   int
	from  int // the first year it exists (0 for always)
}

var rules = []rule{
	{name: "New Year's Day", month: time.January, day: 1},
	{name: "Martin Luther King Jr. Day", month: time.January, wd: time.Monday, nth: 3},
	{name: "Presidents' Day", month: time.February, wd: time.Monday, nth: 3},
	{name: "Memorial Day", month: time.May, wd: time.Monday, nth: -1},
	{name: "Juneteenth", month: time.June, day: 19, from: 2021},
	{name: "Independence Day", month: time.July, day: 4},
	{name: "Labor Day", month: time.September, wd: time.Monday, nth: 1},
	{name: "Columbus Day", month: time.October, wd: time.Monday, nth: 2},
	{name: "Veterans Day", month: time.November, day: 11},
	{name: "Thanksgiving Day", month: time.November, wd: time.Thursday, nth: 4},
	{name: "Christmas Day", month: time.December, day: 25},
}

// dateIn is the rule's date in a year, at midnight UTC.
func (r rule) dateIn(year int) time.Time {
	if r.day != 0 {
		return time.Date(year, r.month, r.day, 0, 0, 0, 0, time.UTC)
	}
	if r.nth > 0 {
		first := time.Date(year, r.month, 1, 0, 0, 0, 0, time.UTC)
		offset := (int(r.wd) - int(first.Weekday()) + 7) % 7
		return first.AddDate(0, 0, offset+7*(r.nth-1))
	}
	last := time.Date(year, r.month+1, 0, 0, 0, 0, 0, time.UTC)
	offset := (int(last.Weekday()) - int(r.wd) + 7) % 7
	return last.AddDate(0, 0, -offset)
}

// inYear is every day the holiday puts on the calendar for a year: the day itself,
// and the observed day when it falls at the weekend. The observed Friday can be in
// the year before (1 Jan 2028 gives Fri 31 Dec 2027).
func (r rule) inYear(year int) []Day {
	if year < r.from {
		return nil
	}
	d := r.dateIn(year)
	switch d.Weekday() {
	case time.Saturday:
		return []Day{
			{Date: d.AddDate(0, 0, -1), Name: r.name + " (observed)", Kind: ObservedOpen},
			{Date: d, Name: r.name, Kind: Closed},
		}
	case time.Sunday:
		return []Day{
			{Date: d, Name: r.name, Kind: Sunday},
			{Date: d.AddDate(0, 0, 1), Name: r.name + " (observed)", Kind: Closed},
		}
	}
	return []Day{{Date: d, Name: r.name, Kind: Closed}}
}

// civil is the calendar date of t in its own zone, as midnight UTC, so a range
// compares by date whatever the time of day or zone it was given in.
func civil(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Days returns every holiday day from from to to, both included, by calendar date,
// in date order. Holidays are worked out for the year before from's to the year
// after to's, so an observed Friday in the year before is never missed. The dates
// are midnight UTC; use their Year, Month and Day.
func Days(from, to time.Time) []Day {
	lo, hi := civil(from), civil(to)
	if hi.Before(lo) {
		return nil
	}
	var out []Day
	for year := lo.Year() - 1; year <= hi.Year()+1; year++ {
		for _, r := range rules {
			for _, d := range r.inYear(year) {
				if !d.Date.Before(lo) && !d.Date.After(hi) {
					out = append(out, d)
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}
