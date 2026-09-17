package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stas-comp/comphq"
)

// openTestDBWithMeta is openTestDB (migrate_test.go) plus the "app"
// section's real migrations, since app_meta (where backup status lives)
// is defined there rather than in the ad hoc "test" section the other
// db tests use.
func openTestDBWithMeta(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB := openTestDB(t)
	migrations, err := LoadMigrations(comphq.Migrations, "app")
	if err != nil {
		t.Fatalf("LoadMigrations(app): %v", err)
	}
	if err := RunMigrations(sqlDB, "1.0.0", migrations); err != nil {
		t.Fatalf("RunMigrations(app): %v", err)
	}
	return sqlDB
}

func TestDailyBackupCreatesFileAndRecordsStatus(t *testing.T) {
	sqlDB := openTestDBWithMeta(t)
	dataDir := t.TempDir()
	now := time.Date(2026, 3, 15, 3, 0, 0, 0, time.UTC)

	if err := DailyBackup(sqlDB, dataDir, now); err != nil {
		t.Fatalf("DailyBackup: %v", err)
	}

	path := filepath.Join(dataDir, "backups", "daily", "comphq-2026-03-15.db")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}
	backupDB, err := Open(path)
	if err != nil {
		t.Fatalf("backup file does not open as a SQLite database: %v", err)
	}
	backupDB.Close()

	status, err := GetBackupStatus(sqlDB)
	if err != nil {
		t.Fatalf("GetBackupStatus: %v", err)
	}
	if !status.HasRun || !status.LastBackupOK || status.LastBackupError != "" {
		t.Fatalf("status = %+v, want HasRun+OK with no error", status)
	}
	if !status.LastBackupAt.Equal(now) {
		t.Fatalf("LastBackupAt = %v, want %v", status.LastBackupAt, now)
	}
}

func TestDailyBackupOverwritesSameDayAndPrunesOldest(t *testing.T) {
	sqlDB := openTestDBWithMeta(t)
	dataDir := t.TempDir()
	base := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)

	// Running twice on the same day must overwrite, not add a second file.
	if err := DailyBackup(sqlDB, dataDir, base); err != nil {
		t.Fatalf("DailyBackup (1st): %v", err)
	}
	if err := DailyBackup(sqlDB, dataDir, base); err != nil {
		t.Fatalf("DailyBackup (2nd, same day): %v", err)
	}

	dir := filepath.Join(dataDir, "backups", "daily")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) after two same-day backups = %d, want 1", len(entries))
	}

	// Run enough additional days to exceed the retention count, then check
	// only the newest DailyBackupsToKeep remain and the oldest is gone.
	for i := 1; i <= DailyBackupsToKeep+5; i++ {
		day := base.AddDate(0, 0, i)
		if err := DailyBackup(sqlDB, dataDir, day); err != nil {
			t.Fatalf("DailyBackup (day %d): %v", i, err)
		}
	}

	entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != DailyBackupsToKeep {
		t.Fatalf("len(entries) = %d, want %d", len(entries), DailyBackupsToKeep)
	}
	if _, err := os.Stat(filepath.Join(dir, "comphq-2026-01-01.db")); !os.IsNotExist(err) {
		t.Fatalf("oldest backup (day 0) should have been pruned, stat err = %v", err)
	}
}

func TestPreUpdateBackupUsesSeparateDirAndSmallerRetention(t *testing.T) {
	sqlDB := openTestDBWithMeta(t)
	dataDir := t.TempDir()
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < PreUpdateBackupsToKeep+3; i++ {
		at := base.Add(time.Duration(i) * time.Hour)
		if err := PreUpdateBackup(sqlDB, dataDir, "1.2.3", at); err != nil {
			t.Fatalf("PreUpdateBackup (run %d): %v", i, err)
		}
	}

	dailyDir := filepath.Join(dataDir, "backups", "daily")
	if _, err := os.Stat(dailyDir); !os.IsNotExist(err) {
		t.Fatalf("PreUpdateBackup must not touch the daily dir, stat err = %v", err)
	}

	preDir := filepath.Join(dataDir, "backups", "pre-update")
	entries, err := os.ReadDir(preDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != PreUpdateBackupsToKeep {
		t.Fatalf("len(entries) = %d, want %d", len(entries), PreUpdateBackupsToKeep)
	}
}

func TestRecordBackupResultKeepsLastSuccessTimeAfterALaterFailure(t *testing.T) {
	sqlDB := openTestDBWithMeta(t)
	succeededAt := time.Date(2026, 5, 1, 3, 0, 0, 0, time.UTC)

	if err := recordBackupResult(sqlDB, succeededAt, nil); err != nil {
		t.Fatalf("recordBackupResult (success): %v", err)
	}
	failedAt := succeededAt.Add(24 * time.Hour)
	if err := recordBackupResult(sqlDB, failedAt, os.ErrPermission); err != nil {
		t.Fatalf("recordBackupResult (failure): %v", err)
	}

	status, err := GetBackupStatus(sqlDB)
	if err != nil {
		t.Fatalf("GetBackupStatus: %v", err)
	}
	if status.LastBackupOK {
		t.Fatal("LastBackupOK = true, want false after the later failed attempt")
	}
	if status.LastBackupError == "" {
		t.Fatal("LastBackupError is empty, want the failure recorded")
	}
	if !status.LastBackupAt.Equal(succeededAt) {
		t.Fatalf("LastBackupAt = %v, want it to stay at the last SUCCESSFUL attempt %v", status.LastBackupAt, succeededAt)
	}
}

func TestBackupStatusStaleBeforeAnyBackup(t *testing.T) {
	var status BackupStatus
	if !status.Stale(time.Now()) {
		t.Fatal("a status with no successful backup ever must be stale")
	}
	if !status.Stale24h(time.Now()) {
		t.Fatal("a status with no successful backup ever must be stale24h")
	}
}

func TestBackupStatusStaleThresholds(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	fresh := BackupStatus{HasRun: true, LastBackupAt: now.Add(-1 * time.Hour)}
	if fresh.Stale(now) {
		t.Fatal("1h old backup should not be stale (banner threshold is 48h)")
	}

	old := BackupStatus{HasRun: true, LastBackupAt: now.Add(-49 * time.Hour)}
	if !old.Stale(now) {
		t.Fatal("49h old backup should be stale (banner threshold is 48h)")
	}

	justOver24h := BackupStatus{HasRun: true, LastBackupAt: now.Add(-25 * time.Hour)}
	if !justOver24h.Stale24h(now) {
		t.Fatal("25h old backup should be stale24h (scheduler startup threshold is 24h)")
	}
	within24h := BackupStatus{HasRun: true, LastBackupAt: now.Add(-1 * time.Hour)}
	if within24h.Stale24h(now) {
		t.Fatal("1h old backup should not be stale24h")
	}
}

func TestHasExistingDataFalseOnFreshDatabase(t *testing.T) {
	sqlDB := openTestDB(t) // no migrations applied at all
	existing, err := HasExistingData(sqlDB)
	if err != nil {
		t.Fatalf("HasExistingData: %v", err)
	}
	if existing {
		t.Fatal("HasExistingData = true on a database with no migrations ever applied")
	}
}

func TestHasExistingDataTrueOnceAppMigrationsHaveRun(t *testing.T) {
	sqlDB := openTestDBWithMeta(t)
	existing, err := HasExistingData(sqlDB)
	if err != nil {
		t.Fatalf("HasExistingData: %v", err)
	}
	if !existing {
		t.Fatal("HasExistingData = false once the app section's migrations have run")
	}
}

func TestSetLastBackupAtForTest(t *testing.T) {
	sqlDB := openTestDBWithMeta(t)
	staleTime := time.Now().Add(-72 * time.Hour)

	if err := SetLastBackupAtForTest(sqlDB, staleTime); err != nil {
		t.Fatalf("SetLastBackupAtForTest: %v", err)
	}

	status, err := GetBackupStatus(sqlDB)
	if err != nil {
		t.Fatalf("GetBackupStatus: %v", err)
	}
	if !status.HasRun {
		t.Fatal("HasRun = false after SetLastBackupAtForTest")
	}
	if !status.Stale(time.Now()) {
		t.Fatal("a 72h-old timestamp should read as stale")
	}
}
