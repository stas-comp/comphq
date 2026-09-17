package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return sqlDB
}

func appliedVersions(t *testing.T, sqlDB *sql.DB, section string) []int {
	t.Helper()
	rows, err := sqlDB.Query(`SELECT version FROM schema_migrations WHERE section = ? ORDER BY version`, section)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, v)
	}
	return out
}

func TestRunMigrationsAppliesInOrder(t *testing.T) {
	sqlDB := openTestDB(t)

	migrations := []Migration{
		{Section: "test", Version: 1, Name: "create", SQL: `CREATE TABLE widgets (id INTEGER PRIMARY KEY, name TEXT)`},
		{Section: "test", Version: 2, Name: "add_column", SQL: `ALTER TABLE widgets ADD COLUMN size TEXT`},
	}

	if err := RunMigrations(sqlDB, "1.0.0", migrations); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	got := appliedVersions(t, sqlDB, "test")
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("applied versions = %v, want [1 2]", got)
	}

	// Both migrations actually ran: inserting a row using the column added
	// by migration 2 must succeed.
	if _, err := sqlDB.Exec(`INSERT INTO widgets (name, size) VALUES ('a', 'M')`); err != nil {
		t.Fatalf("insert after migrations: %v", err)
	}

	// Running again must be a no-op, not a re-apply (which would fail on
	// CREATE TABLE / duplicate ALTER).
	if err := RunMigrations(sqlDB, "1.0.0", migrations); err != nil {
		t.Fatalf("RunMigrations (second run): %v", err)
	}
}

func TestRunMigrationsIgnoresUnknownNewerRows(t *testing.T) {
	sqlDB := openTestDB(t)

	// Simulate a newer app having already applied a migration this older
	// app doesn't have a file for.
	if err := ensureMigrationsTable(sqlDB); err != nil {
		t.Fatalf("ensureMigrationsTable: %v", err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO schema_migrations (section, version, name, applied_at, app_version) VALUES ('test', 99, 'future', '2026-01-01T00:00:00Z', '9.9.9')`,
	); err != nil {
		t.Fatalf("seed future row: %v", err)
	}

	migrations := []Migration{
		{Section: "test", Version: 1, Name: "create", SQL: `CREATE TABLE widgets (id INTEGER PRIMARY KEY)`},
	}

	if err := RunMigrations(sqlDB, "1.0.0", migrations); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	got := appliedVersions(t, sqlDB, "test")
	if len(got) != 2 || got[0] != 1 || got[1] != 99 {
		t.Fatalf("applied versions = %v, want [1 99] (the unknown row must be left alone)", got)
	}
}

func TestRunMigrationsRefusesOnFailedMigration(t *testing.T) {
	sqlDB := openTestDB(t)

	migrations := []Migration{
		{Section: "test", Version: 1, Name: "broken", SQL: `THIS IS NOT VALID SQL`},
	}

	if err := RunMigrations(sqlDB, "1.0.0", migrations); err == nil {
		t.Fatal("RunMigrations: want an error for invalid SQL, got nil")
	}

	got := appliedVersions(t, sqlDB, "test")
	if len(got) != 0 {
		t.Fatalf("applied versions = %v, want none recorded after a failed migration", got)
	}
}

func TestHasPendingMigrations(t *testing.T) {
	sqlDB := openTestDB(t)
	migrations := []Migration{
		{Section: "test", Version: 1, Name: "create", SQL: `CREATE TABLE widgets (id INTEGER PRIMARY KEY)`},
		{Section: "test", Version: 2, Name: "add_column", SQL: `ALTER TABLE widgets ADD COLUMN size TEXT`},
	}

	pending, err := HasPendingMigrations(sqlDB, migrations)
	if err != nil {
		t.Fatalf("HasPendingMigrations (before any run): %v", err)
	}
	if !pending {
		t.Fatal("pending = false, want true before any migration has been applied")
	}

	if err := RunMigrations(sqlDB, "1.0.0", migrations); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	pending, err = HasPendingMigrations(sqlDB, migrations)
	if err != nil {
		t.Fatalf("HasPendingMigrations (after run): %v", err)
	}
	if pending {
		t.Fatal("pending = true, want false once every migration has been applied")
	}

	migrations = append(migrations, Migration{Section: "test", Version: 3, Name: "add_index", SQL: `CREATE INDEX idx_widgets_size ON widgets(size)`})
	pending, err = HasPendingMigrations(sqlDB, migrations)
	if err != nil {
		t.Fatalf("HasPendingMigrations (with a new migration appended): %v", err)
	}
	if !pending {
		t.Fatal("pending = false, want true once a new migration file is added")
	}
}

func TestLoadMigrationsSortsByVersion(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/test/0002_second.sql": {Data: []byte("-- second\n")},
		"migrations/test/0001_first.sql":  {Data: []byte("-- first\n")},
	}

	migrations, err := LoadMigrations(fsys, "test")
	if err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	if len(migrations) != 2 {
		t.Fatalf("len(migrations) = %d, want 2", len(migrations))
	}
	if migrations[0].Version != 1 || migrations[0].Name != "first" {
		t.Errorf("migrations[0] = %+v, want version 1 name first", migrations[0])
	}
	if migrations[1].Version != 2 || migrations[1].Name != "second" {
		t.Errorf("migrations[1] = %+v, want version 2 name second", migrations[1])
	}
}
