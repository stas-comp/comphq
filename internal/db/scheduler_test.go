package db

import (
	"database/sql"
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

// waitForBackupStatus polls GetBackupStatus rather than the daily
// backup file's mere existence: backupAndRecord writes the file (via
// VACUUM INTO + rename) and then records app_meta as two separate
// steps, so a test that only waited for the file could read app_meta in
// the narrow window before that second step lands (caught for real on
// CI's Linux runner, where the whole test finished in under 10ms).
func waitForBackupStatus(t *testing.T, sqlDB *sql.DB, timeout time.Duration) BackupStatus {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last BackupStatus
	for time.Now().Before(deadline) {
		status, err := GetBackupStatus(sqlDB)
		if err != nil {
			t.Fatalf("GetBackupStatus: %v", err)
		}
		if status.HasRun {
			return status
		}
		last = status
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no backup status recorded within %v (last read: %+v)", timeout, last)
	return BackupStatus{}
}

func TestRunBackupSchedulerRunsImmediatelyWhenStale(t *testing.T) {
	sqlDB := openTestDBWithMeta(t)
	dataDir := t.TempDir()
	clock := blockingClock{now: time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)}

	go RunBackupScheduler(sqlDB, dataDir, clock)

	status := waitForBackupStatus(t, sqlDB, 2*time.Second)
	if !status.LastBackupOK {
		t.Fatalf("status = %+v, want a successful backup recorded", status)
	}

	dir := filepath.Join(dataDir, "backups", "daily")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no daily backup file found in %s (err: %v)", dir, err)
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
