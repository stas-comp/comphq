package calendar

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
)

var recurrenceLabels = map[string]string{
	RecurrenceNone:    "Never",
	RecurrenceWeekly:  "Weekly",
	RecurrenceMonthly: "Monthly",
	RecurrenceYearly:  "Yearly",
}

// ExportRows returns every event, removed ones included, as rows of
// plain text for events.csv (SPEC B5, gate 2.22): a header row in plain
// English, then one row per event in creation order, with dates the way
// SPEC A3 writes them ("Sat 19 Sep 2026"). Bare [][]string, so
// internal/settings can consume it through a one-method interface
// without importing this package (SPEC B2, D-53).
func (s *Store) ExportRows(ctx context.Context) ([][]string, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT title, notes, start_date, end_date, start_time, end_time, recurrence, until_date, notice_days, removed_at
		FROM events ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := [][]string{{"Title", "Date", "End date", "Start time", "End time", "Repeats", "Until", "Show ahead", "Notes", "Removed"}}
	for rows.Next() {
		var title, notes, start, recurrence string
		var endDate, startTime, endTime, until, removedAt sql.NullString
		var notice int
		if err := rows.Scan(&title, &notes, &start, &endDate, &startTime, &endTime, &recurrence, &until, &notice, &removedAt); err != nil {
			return nil, err
		}
		removed := ""
		if removedAt.String != "" {
			if t, err := time.Parse(time.RFC3339, removedAt.String); err == nil {
				removed = format.DateTime(t)
			} else {
				removed = removedAt.String
			}
		}
		out = append(out, []string{
			title, exportDate(start), exportDate(endDate.String), startTime.String, endTime.String,
			recurrenceLabels[recurrence], exportDate(until.String), showAheadLabel(notice), notes, removed,
		})
	}
	return out, rows.Err()
}

func exportDate(s string) string {
	if s == "" {
		return ""
	}
	if t, err := time.Parse(dateLayout, s); err == nil {
		return format.Date(t)
	}
	return s
}

// showAheadLabel words notice_days as the form does (SPEC B3: weeks are
// stored as days x7 and shown in weeks when evenly divisible).
func showAheadLabel(days int) string {
	amount, unit := noticeAmountUnit(days)
	if amount == 0 {
		return "Not early"
	}
	word := unit[:len(unit)-1] // "weeks" -> "week"
	if amount == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %s", amount, unit)
}
