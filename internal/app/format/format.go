// Package format renders dates and times the way SPEC A3 requires
// everywhere in the app: "Sat 19 Sep 2026" and "14:30".
package format

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Date renders t as "Sat 19 Sep 2026".
func Date(t time.Time) string {
	return t.Format("Mon 2 Jan 2006")
}

// Clock renders t as "14:30".
func Clock(t time.Time) string {
	return t.Format("15:04")
}

// DateTime renders t as "Sat 19 Sep 2026, 14:30".
func DateTime(t time.Time) string {
	return Date(t) + ", " + Clock(t)
}

// dayFirstRe matches a date typed the way this office writes it: day,
// month, year, separated by "/", "-" or ".", with the day and month
// optionally without their leading zero ("5/9/2026").
var dayFirstRe = regexp.MustCompile(`^(\d{1,2})[/.-](\d{1,2})[/.-](\d{4})$`)

// ParseDayFirst reads a typed date in day-first order ("25/09/2026") and
// returns it as the ISO "2026-09-25" the app stores (SPEC gate 4.05: the
// date field shows and accepts UK order). An already-ISO date is accepted
// as it is, so a script or an old bookmark that sends "2026-09-25" keeps
// working. ok is false for anything else, including a day that doesn't
// exist (31/02/2026).
func ParseDayFirst(s string) (iso string, ok bool) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Format("2006-01-02"), true
	}
	m := dayFirstRe.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	t, err := time.Parse("2006-01-02", fmt.Sprintf("%s-%02s-%02s", m[3], m[2], m[1]))
	if err != nil {
		return "", false
	}
	return t.Format("2006-01-02"), true
}

// ValidISODate reports whether s is a real calendar date in the stored
// "2026-09-25" form.
func ValidISODate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// dayFirstNoYearRe matches a date typed without its year ("25/9", "25.9",
// "25-9"): day and month only, the same separators as dayFirstRe.
var dayFirstNoYearRe = regexp.MustCompile(`^(\d{1,2})[/.-](\d{1,2})$`)

// ParseDayFirstNear reads everything ParseDayFirst does, plus a date typed
// without its year (SPEC B12.2, D-82): "25/9" means the 25 September
// nearest to today. It tries the day and month in last year, this year and
// next year, and keeps whichever real date is closest to today (ties are
// impossible: candidates are exactly a year apart, never an even number of
// days). A day that doesn't exist in one of those years (29 February) is
// simply not a candidate, so 29/2 still resolves to the nearest real leap
// year rather than being refused outright.
func ParseDayFirstNear(s string, today time.Time) (iso string, ok bool) {
	s = strings.TrimSpace(s)
	if iso, ok := ParseDayFirst(s); ok {
		return iso, ok
	}
	m := dayFirstNoYearRe.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	day, month := m[1], m[2]

	var best time.Time
	var bestDiff time.Duration
	found := false
	for _, year := range [3]int{today.Year() - 1, today.Year(), today.Year() + 1} {
		t, err := time.Parse("2006-01-02", fmt.Sprintf("%04d-%02s-%02s", year, month, day))
		if err != nil {
			continue // not a real day in this year (29 February)
		}
		diff := t.Sub(today)
		if diff < 0 {
			diff = -diff
		}
		if !found || diff < bestDiff {
			best, bestDiff, found = t, diff, true
		}
	}
	if !found {
		return "", false
	}
	return best.Format("2006-01-02"), true
}

// NormaliseDate turns a typed date — including one without its year,
// SPEC B12.2 — into the stored ISO form when it can, and otherwise returns
// the text unchanged, so the code that validates a date sees exactly what
// was typed and refuses it as it always has. today decides which year a
// yearless date means (D-82); callers pass app.Today(...), so test mode's
// fixed date is honoured everywhere a date is typed.
func NormaliseDate(s string, today time.Time) string {
	if iso, ok := ParseDayFirstNear(s, today); ok {
		return iso
	}
	return strings.TrimSpace(s)
}

// DayFirst renders a stored ISO date the way the date field shows it,
// "25/09/2026". Anything that isn't a date (empty, or text that was typed
// wrongly and is being shown back) is returned unchanged.
func DayFirst(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("02/01/2006")
}

// Initials returns up to two uppercase initials from a person's name, the
// first letter of the first and last words (SPEC A6: "people's initials in
// coloured circles"): "Sam Jones" is "SJ", "Sam" is "S".
func Initials(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	first := func(s string) string {
		for _, r := range s {
			return strings.ToUpper(string(r))
		}
		return ""
	}
	initials := first(fields[0])
	if len(fields) > 1 {
		initials += first(fields[len(fields)-1])
	}
	return initials
}

// DateShort renders t as "Wed 23 Sep" for a card, adding the year only when
// it isn't today's year ("Wed 23 Sep 2027"): a card is too narrow for the
// full A3 form, and a date in the current year needs no year.
func DateShort(t, today time.Time) string {
	if t.Year() == today.Year() {
		return t.Format("Mon 2 Jan")
	}
	return t.Format("Mon 2 Jan 2006")
}
