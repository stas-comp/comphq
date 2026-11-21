package app

import (
	"log"
	"os"
	"time"
)

const testTodayEnv = "COMPHQ_TEST_TODAY"

// Today returns today's local date at midnight, honouring
// COMPHQ_TEST_TODAY in test mode (SPEC B4) so E2E tests can assert
// date-dependent behaviour — an overdue task, the Saturday Briefing —
// without waiting for a real date to arrive. Never set in
// deploy/truenas.yaml (tools/policy's TestDeployYAMLRules checks).
func Today(testMode bool) time.Time {
	if testMode {
		if v := os.Getenv(testTodayEnv); v != "" {
			if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
				log.Printf("WARNING: %s=%s is overriding today's date (test mode only)", testTodayEnv, v)
				return t
			}
		}
	}
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// Now is the moment a change is made, for timestamps that must tell two
// changes on the same day apart (the finishing time that orders Done, D-104).
// It is the real clock, except that COMPHQ_TEST_TODAY in test mode keeps the
// date it names and takes only the time of day from the real clock, so a test
// that fixes the date still sees every day-based rule (the 14-day tuck-away,
// the weekly and yearly passes) behave as if it were that day.
func Now(testMode bool) time.Time {
	real := time.Now()
	if !testMode {
		return real
	}
	today := Today(true)
	if today.Year() == real.Year() && today.YearDay() == real.YearDay() {
		return real
	}
	return time.Date(today.Year(), today.Month(), today.Day(), real.Hour(), real.Minute(), real.Second(), real.Nanosecond(), real.Location())
}
