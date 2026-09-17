package db

import (
	"database/sql"
	"log"
	"time"
)

// Clock lets tests substitute a fake time source; production uses realClock.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

// RealClock is the production Clock, backed by the real wall clock.
type RealClock struct{}

func (RealClock) Now() time.Time                         { return time.Now() }
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// RunBackupScheduler runs DailyBackup once at startup if none has
// succeeded in the last 24h, then once every day at 03:00 local time
// (SPEC B5). It never returns; call it in its own goroutine. Errors are
// logged (backupAndRecord already saves them to app_meta for the banner
// and Settings page) rather than stopping the loop, since a single
// failed night shouldn't cancel every night after it.
func RunBackupScheduler(sqlDB *sql.DB, dataDir string, clock Clock) {
	if status, err := GetBackupStatus(sqlDB); err != nil {
		log.Printf("backup scheduler: reading backup status: %v", err)
	} else if status.Stale24h(clock.Now()) {
		runScheduledBackup(sqlDB, dataDir, clock)
	}

	for {
		wait := nextDailyBackupIn(clock.Now())
		<-clock.After(wait)
		runScheduledBackup(sqlDB, dataDir, clock)
	}
}

func runScheduledBackup(sqlDB *sql.DB, dataDir string, clock Clock) {
	if err := DailyBackup(sqlDB, dataDir, clock.Now()); err != nil {
		log.Printf("backup scheduler: daily backup failed: %v", err)
	}
}

// Stale24h is the startup check ("no backup in the last 24h"), distinct
// from Stale's 48h banner threshold.
func (s BackupStatus) Stale24h(now time.Time) bool {
	return !s.HasRun || now.Sub(s.LastBackupAt) >= 24*time.Hour
}

// nextDailyBackupIn computes the duration until the next 03:00 in now's
// own location. Building the target from now's date/location components
// (rather than adding a fixed 24h) keeps this correct across a DST
// transition, when a calendar day is 23 or 25 hours long.
func nextDailyBackupIn(now time.Time) time.Duration {
	loc := now.Location()
	target := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, loc)
	if !target.After(now) {
		target = target.AddDate(0, 0, 1)
	}
	return target.Sub(now)
}
