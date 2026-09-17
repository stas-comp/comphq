package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration is one numbered SQL file under migrations/<section>/.
type Migration struct {
	Section string
	Version int
	Name    string
	SQL     string
}

// LoadMigrations reads migrations/<section>/NNNN_name.sql from fsys, sorted
// by version.
func LoadMigrations(fsys fs.FS, section string) ([]Migration, error) {
	dir := "migrations/" + section
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	migrations := make([]Migration, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		version, name, err := parseMigrationFilename(e.Name())
		if err != nil {
			return nil, fmt.Errorf("%s/%s: %w", dir, e.Name(), err)
		}
		content, err := fs.ReadFile(fsys, dir+"/"+e.Name())
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, Migration{
			Section: section,
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}

func parseMigrationFilename(filename string) (version int, name string, err error) {
	base := strings.TrimSuffix(filename, ".sql")
	prefix, rest, ok := strings.Cut(base, "_")
	if !ok {
		return 0, "", fmt.Errorf("expected NNNN_name.sql")
	}
	version, err = strconv.Atoi(prefix)
	if err != nil {
		return 0, "", fmt.Errorf("expected a numeric prefix: %w", err)
	}
	return version, rest, nil
}

// RunMigrations applies every migration not yet recorded in
// schema_migrations, in order, each in its own transaction. Rows already in
// schema_migrations for versions this app has no file for (applied by a
// newer app) are left untouched and never re-examined.
func RunMigrations(sqlDB *sql.DB, appVersion string, migrations []Migration) error {
	if err := ensureMigrationsTable(sqlDB); err != nil {
		return fmt.Errorf("prepare schema_migrations: %w", err)
	}
	for _, m := range migrations {
		applied, err := isApplied(sqlDB, m.Section, m.Version)
		if err != nil {
			return fmt.Errorf("check %s/%04d_%s: %w", m.Section, m.Version, m.Name, err)
		}
		if applied {
			continue
		}
		if err := applyMigration(sqlDB, appVersion, m); err != nil {
			return fmt.Errorf("migration %s/%04d_%s: %w", m.Section, m.Version, m.Name, err)
		}
	}
	return nil
}

// HasPendingMigrations reports whether any migration in the list has not
// yet been recorded in schema_migrations (SPEC B6: the pre-update backup
// runs "only when migrations are pending").
func HasPendingMigrations(sqlDB *sql.DB, migrations []Migration) (bool, error) {
	if err := ensureMigrationsTable(sqlDB); err != nil {
		return false, fmt.Errorf("prepare schema_migrations: %w", err)
	}
	for _, m := range migrations {
		applied, err := isApplied(sqlDB, m.Section, m.Version)
		if err != nil {
			return false, fmt.Errorf("check %s/%04d_%s: %w", m.Section, m.Version, m.Name, err)
		}
		if !applied {
			return true, nil
		}
	}
	return false, nil
}

func ensureMigrationsTable(sqlDB *sql.DB) error {
	_, err := sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			section     TEXT NOT NULL,
			version     INTEGER NOT NULL,
			name        TEXT NOT NULL,
			applied_at  TEXT NOT NULL,
			app_version TEXT NOT NULL,
			PRIMARY KEY (section, version)
		)
	`)
	return err
}

func isApplied(sqlDB *sql.DB, section string, version int) (bool, error) {
	var count int
	err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE section = ? AND version = ?`,
		section, version,
	).Scan(&count)
	return count > 0, err
}

func applyMigration(sqlDB *sql.DB, appVersion string, m Migration) error {
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(m.SQL); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO schema_migrations (section, version, name, applied_at, app_version) VALUES (?, ?, ?, ?, ?)`,
		m.Section, m.Version, m.Name, time.Now().UTC().Format(time.RFC3339), appVersion,
	); err != nil {
		return err
	}
	return tx.Commit()
}
