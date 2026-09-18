package tasks

import (
	"context"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
)

// ExportRows returns every task, removed ones included, as rows of
// plain text for tasks.csv (SPEC B5, gate 2.22): a header row in plain
// English, then one row per task in creation order, with dates the way
// SPEC A3 writes them everywhere ("Sat 19 Sep 2026"). It returns bare
// [][]string, not a task-specific type, so internal/settings can
// consume it through a one-method interface without importing this
// package (SPEC B2, D-53).
func (s *Store) ExportRows(ctx context.Context) ([][]string, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT t.id, t.title, t.stage, t.size, t.due_date, p.name, t.created_at, t.done_at, t.removed_at
		FROM tasks t JOIN people p ON p.id = t.created_by
		ORDER BY t.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type raw struct {
		id                                     int64
		title, stage, size, creator, createdAt string
		dueDate, doneAt, removedAt             *string
	}
	var all []raw
	var ids []int64
	for rows.Next() {
		var r raw
		if err := rows.Scan(&r.id, &r.title, &r.stage, &r.size, &r.dueDate, &r.creator, &r.createdAt, &r.doneAt, &r.removedAt); err != nil {
			return nil, err
		}
		all = append(all, r)
		ids = append(ids, r.id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	assignees, err := s.assigneesFor(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := [][]string{{"Title", "Column", "Size", "People", "Due date", "Created by", "Created", "Finished", "Removed"}}
	for _, r := range all {
		var people []string
		for _, a := range assignees[r.id] {
			people = append(people, a.Name)
		}
		due := ""
		if r.dueDate != nil {
			due = exportDate(*r.dueDate)
		}
		out = append(out, []string{
			r.title, stageLabels[r.stage], sizeLabels[r.size], strings.Join(people, ", "),
			due, r.creator, exportDateTime(r.createdAt), exportDateTimePtr(r.doneAt), exportDateTimePtr(r.removedAt),
		})
	}
	return out, nil
}

func exportDate(s string) string {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return format.Date(t)
	}
	return s
}

func exportDateTime(s string) string {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return format.DateTime(t)
	}
	return s
}

func exportDateTimePtr(s *string) string {
	if s == nil || *s == "" {
		return ""
	}
	return exportDateTime(*s)
}
