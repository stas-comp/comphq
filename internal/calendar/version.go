package calendar

import (
	"context"
	"database/sql"
	"strconv"
)

// calendarVersionKey is the app_meta row this package's change counter
// lives under, the Calendar's counterpart to tasks_version (SPEC B4's
// refresh design, D-15). Every write bumps it in the same transaction
// as the write itself, so it only advances if the change committed;
// the Briefing polls it, with tasks_version, to notice changes made
// elsewhere (SPEC gate 3.12).
const calendarVersionKey = "calendar_version"

// inTx runs write inside a transaction that also bumps calendar_version.
// write's own error (including ErrEventNotFound) rolls everything back,
// counter included. Nothing else may query the database from inside
// write: the single connection is held by the transaction (D-44).
func (s *Store) inTx(ctx context.Context, write func(tx *sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := write(tx); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO app_meta (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)
	`, calendarVersionKey); err != nil {
		return err
	}
	return tx.Commit()
}

// expectOneRow turns an UPDATE's result into ErrEventNotFound when it
// matched no row.
func expectOneRow(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrEventNotFound
	}
	return nil
}

// Version returns the current counter, 0 if no write has ever bumped
// it (a fresh install with no events yet).
func (s *Store) Version(ctx context.Context) (int64, error) {
	var value sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, calendarVersionKey).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(value.String, 10, 64)
}
