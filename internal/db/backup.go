package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// DailyBackupsToKeep and PreUpdateBackupsToKeep are SPEC B5's retention
// counts: a rolling window of recovery points for the daily copies, and a
// much shorter one for pre-update copies, which exist to protect one
// specific upgrade rather than give ongoing coverage.
const (
	DailyBackupsToKeep     = 30
	PreUpdateBackupsToKeep = 10
)

// BackupStatus is what Settings and the stale-backup banner read (SPEC
// B3: "last_backup_at, last_backup_ok, last_backup_error" in app_meta).
// LastBackupAt only ever advances on a successful backup, so it always
// answers "when did a backup last actually succeed" even if a later
// attempt failed; LastBackupOK/LastBackupError describe that later
// attempt's own outcome. HasRun is false until the very first backup.
type BackupStatus struct {
	HasRun          bool
	LastBackupAt    time.Time
	LastBackupOK    bool
	LastBackupError string
}

// Stale reports whether the banner should show (SPEC gate 1.35: "now -
// last successful backup > 48h" — including never having had one).
func (s BackupStatus) Stale(now time.Time) bool {
	return !s.HasRun || now.Sub(s.LastBackupAt) > 48*time.Hour
}

// DailyBackup runs the nightly backup (SPEC B5). comphq-YYYY-MM-DD.db
// filenames sort chronologically as plain strings, so pruning by name is
// pruning by date; running twice in the same day overwrites that day's
// file, which is exactly "one backup per day."
func DailyBackup(sqlDB *sql.DB, dataDir string, now time.Time) error {
	dir := filepath.Join(dataDir, "backups", "daily")
	filename := fmt.Sprintf("comphq-%s.db", now.UTC().Format("2006-01-02"))
	return backupAndRecord(sqlDB, dir, filename, DailyBackupsToKeep, now)
}

// PreUpdateBackup runs the pre-migration backup (SPEC B4: "after a
// VACUUM INTO pre-update backup, only when migrations are pending").
func PreUpdateBackup(sqlDB *sql.DB, dataDir, appVersion string, now time.Time) error {
	dir := filepath.Join(dataDir, "backups", "pre-update")
	filename := fmt.Sprintf("comphq-v%s-%s.db", appVersion, now.UTC().Format("20060102T150405Z"))
	return backupAndRecord(sqlDB, dir, filename, PreUpdateBackupsToKeep, now)
}

func backupAndRecord(sqlDB *sql.DB, dir, filename string, keep int, now time.Time) error {
	err := backupInto(sqlDB, dir, filename, keep)
	if recordErr := recordBackupResult(sqlDB, now, err); recordErr != nil && err == nil {
		return recordErr
	}
	return err
}

// backupInto runs VACUUM INTO to a temporary name in dir, renames it into
// place only once fully written (so nothing ever reads a half-written
// file), then prunes dir to the newest keep entries.
func backupInto(sqlDB *sql.DB, dir, filename string, keep int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(dir, "."+filename+".tmp")
	target := filepath.Join(dir, filename)
	os.Remove(tmp) // a leftover from a previous crash would make VACUUM INTO refuse to write

	if _, err := sqlDB.Exec(`VACUUM INTO ?`, tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		return err
	}
	return pruneOldest(dir, keep)
}

func pruneOldest(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) <= keep {
		return nil
	}
	for _, name := range names[:len(names)-keep] {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

func recordBackupResult(sqlDB *sql.DB, at time.Time, backupErr error) error {
	kv := map[string]string{"last_backup_ok": "1", "last_backup_error": ""}
	if backupErr != nil {
		kv["last_backup_ok"] = "0"
		kv["last_backup_error"] = backupErr.Error()
	} else {
		kv["last_backup_at"] = at.UTC().Format(time.RFC3339)
	}
	return setAppMeta(sqlDB, kv)
}

func setAppMeta(sqlDB *sql.DB, kv map[string]string) error {
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for k, v := range kv {
		if _, err := tx.Exec(
			`INSERT INTO app_meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			k, v,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetBackupStatus reads what DailyBackup/PreUpdateBackup last recorded.
func GetBackupStatus(sqlDB *sql.DB) (BackupStatus, error) {
	rows, err := sqlDB.Query(`SELECT key, value FROM app_meta WHERE key IN ('last_backup_at', 'last_backup_ok', 'last_backup_error')`)
	if err != nil {
		return BackupStatus{}, err
	}
	defer rows.Close()

	var status BackupStatus
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return BackupStatus{}, err
		}
		switch key {
		case "last_backup_at":
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				continue
			}
			status.LastBackupAt = t
			status.HasRun = true
		case "last_backup_ok":
			status.LastBackupOK = value == "1"
		case "last_backup_error":
			status.LastBackupError = value
		}
	}
	if err := rows.Err(); err != nil {
		return BackupStatus{}, err
	}
	return status, nil
}

// HasExistingData reports whether this database has ever had the "app"
// section's migrations run against it, by checking for app_meta (created
// by that section's very first migration). A fresh install has neither
// the table nor anything worth protecting with a pre-update backup — and
// backupAndRecord couldn't record a result into a table that doesn't
// exist yet — so callers should skip PreUpdateBackup when this is false.
func HasExistingData(sqlDB *sql.DB) (bool, error) {
	var name string
	err := sqlDB.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'app_meta'`).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetLastBackupAtForTest directly sets last_backup_at (test-mode only —
// SPEC gate 1.35's E2E banner test: "set last_backup_at to 3 days ago"),
// bypassing an actual backup entirely.
func SetLastBackupAtForTest(sqlDB *sql.DB, at time.Time) error {
	return setAppMeta(sqlDB, map[string]string{
		"last_backup_at": at.UTC().Format(time.RFC3339),
		"last_backup_ok": "1",
	})
}
