// Package db opens the Comp HQ SQLite database and runs section migrations.
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens the SQLite database at path with the pragmas SPEC B1 requires:
// WAL mode, foreign keys on, and a busy timeout so concurrent access waits
// instead of failing immediately.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)",
		path,
	)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	// SQLite has one writer at a time; a single connection avoids
	// "database is locked" errors under the driver's own retry logic.
	sqlDB.SetMaxOpenConns(1)
	return sqlDB, nil
}
