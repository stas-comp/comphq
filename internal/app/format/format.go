// Package format renders dates and times the way SPEC A3 requires
// everywhere in the app: "Sat 19 Sep 2026" and "14:30".
package format

import "time"

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
