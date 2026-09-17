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
