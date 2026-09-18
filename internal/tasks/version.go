package tasks

import (
	"context"
	"database/sql"
	"strconv"
)

// tasksVersionKey is the app_meta row this package's own change counter
// lives under (SPEC B4's refresh design, PLAN.md P2-07/D-15): every
// write bumps it in the same transaction, and GET /tasks/version lets a
// client cheaply notice a change happened elsewhere without re-fetching
// the whole board on a timer.
const tasksVersionKey = "tasks_version"

// bumpTasksVersion increments the counter (starting at 1 if unset)
// within the caller's own transaction, so it only actually advances if
// that transaction commits.
func bumpTasksVersion(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO app_meta (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)
	`, tasksVersionKey)
	return err
}

// Version returns the current counter, 0 if nothing has ever bumped it
// (a fresh install with no tasks yet).
func (s *Store) Version(ctx context.Context) (int64, error) {
	var value sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, tasksVersionKey).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	version, err := strconv.ParseInt(value.String, 10, 64)
	if err != nil {
		return 0, err
	}
	return version, nil
}
