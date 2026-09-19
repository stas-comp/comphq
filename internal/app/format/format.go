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

// NormaliseDate turns a typed date into the stored ISO form when it can,
// and otherwise returns the text unchanged, so the code that validates a
// date sees exactly what was typed and refuses it as it always has.
func NormaliseDate(s string) string {
	if iso, ok := ParseDayFirst(s); ok {
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
