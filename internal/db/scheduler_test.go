package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Skipf("tzdata for %q not available in this environment: %v", name, err)
	}
	return loc
}

func TestNextDailyBackupInBeforeTarget(t *testing.T) {
	now := time.Date(2026, 1, 10, 1, 0, 0, 0, time.UTC)
	if got, want := nextDailyBackupIn(now), 2*time.Hour; got != want {
		t.Fatalf("nextDailyBackupIn = %v, want %v", got, want)
	}
}

func TestNextDailyBackupInAfterTargetRollsToTomorrow(t *testing.T) {
	now := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	if got, want := nextDailyBackupIn(now), 17*time.Hour; got != want {
		t.Fatalf("nextDailyBackupIn = %v, want %v", got, want)
	}
}

func TestNextDailyBackupInAtExactlyTargetRollsToTomorrow(t *testing.T) {
	now := time.Date(2026, 1, 10, 3, 0, 0, 0, time.UTC)
	if got, want := nextDailyBackupIn(now), 24*time.Hour; got != want {
		t.Fatalf("nextDailyBackupIn = %v, want %v", got, want)
	}
}

// Europe/London springs forward on 2026-03-29 (clocks jump 01:00 -> 02:00
// local), making that calendar day 23 hours long. Computing the target
// from now's own date/location (rather than adding a fixed 24h) must
// still land on 03:00 local the next day.
func TestNextDailyBackupInAcrossSpringForwardDST(t *testing.T) {
	loc := mustLoadLocation(t, "Europe/London")
	now := time.Date(2026, 3, 28, 20, 0, 0, 0, loc)

	next := now.Add(nextDailyBackupIn(now))

	if next.Hour() != 3 || next.Minute() != 0 {
		t.Fatalf("next backup time = %v, want 03:00 local", next)
	}
	if next.Day() != 29 {
		t.Fatalf("next backup day = %d, want 29 (the day after now)", next.Day())
	}
}

// Europe/London falls back on 2026-10-25 (clocks step 02:00 -> 01:00
// local), making that calendar day 25 hours long.
func TestNextDailyBackupInAcrossFallBackDST(t *testing.T) {
	loc := mustLoadLocation(t, "Europe/London")
	now := time.Date(2026, 10, 24, 20, 0, 0, 0, loc)

	next := now.Add(nextDailyBackupIn(now))

	if next.Hour() != 3 || next.Minute() != 0 {
		t.Fatalf("next backup time = %v, want 03:00 local", next)
	}
	if next.Day() != 25 {
		t.Fatalf("next backup day = %d, want 25 (the day after now)", next.Day())
	}
}

// blockingClock's After never fires, so a scheduler under test runs its
// startup check and then parks forever on the loop's first wait — enough
// to observe RunBackupScheduler's startup decision without needing the
// test to also drive a fake day boundary.
type blockingClock struct{ now time.Time }

func (c blockingClock) Now() time.Time                         { return c.now }
func (c blockingClock) After(time.Duration) <-chan time.Time { return make(chan time.Time) }

func waitForDailyBackupFile(t *testing.T, dataDir string, timeout time.Duration) {
	t.Helper()
	dir := filepath.Join(dataDir, "backups", "daily")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no daily backup appeared in %s within %v", dir, timeout)
}

func TestRunBackupSchedulerRunsImmediatelyWhenStale(t *testing.T) {
	sqlDB := openTestDBWithMeta(t)
	dataDir := t.TempDir()
	clock := blockingClock{now: time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)}

	go RunBackupScheduler(sqlDB, dataDir, clock)

	waitForDailyBackupFile(t, dataDir, 2*time.Second)

	status, err := GetBackupStatus(sqlDB)
	if err != nil {
		t.Fatalf("GetBackupStatus: %v", err)
	}
	if !status.HasRun || !status.LastBackupOK {
		t.Fatalf("status = %+v, want a successful backup recorded", status)
	}
}

func TestRunBackupSchedulerSkipsImmediateRunWhenFresh(t *testing.T) {
	sqlDB := openTestDBWithMeta(t)
	dataDir := t.TempDir()
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	if err := SetLastBackupAtForTest(sqlDB, now.Add(-1*time.Hour)); err != nil {
		t.Fatalf("SetLastBackupAtForTest: %v", err)
	}
	clock := blockingClock{now: now}

	go RunBackupScheduler(sqlDB, dataDir, clock)

	// The scheduler's only synchronous startup work is the stale24h check;
	// give it a generous window to (wrongly) run a backup before asserting
	// it didn't. Once past this window it can only still be idle, since
	// blockingClock's After() never wakes the loop for a second attempt.
	time.Sleep(200 * time.Millisecond)

	dir := filepath.Join(dataDir, "backups", "daily")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("no backup should run at startup with a fresh last_backup_at, stat err = %v", err)
	}
}
