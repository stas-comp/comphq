package tasks

import (
	"context"
	"time"
)

// DueTask is one row of DueTasks' result (SPEC gate 2.19) — Calendar's
// own small query interface onto this package, injected at
// registration rather than importing this package directly (SPEC B2:
// "sections never import each other's internals").
type DueTask struct {
	ID      int64
	Title   string
	DueDate string
}

// DueTasks returns every unfinished, unremoved task with a due date in
// [from, to] (SPEC gate 2.19: "Unfinished tasks with due dates show on
// their date... Finished and removed tasks don't show"), ordered by
// due date then board position.
func (s *Store) DueTasks(ctx context.Context, from, to time.Time) ([]DueTask, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, title, due_date
		FROM tasks
		WHERE removed_at IS NULL
		  AND stage != ?
		  AND due_date IS NOT NULL AND due_date != ''
		  AND due_date BETWEEN ? AND ?
		ORDER BY due_date, position
	`, StageDone, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DueTask
	for rows.Next() {
		var d DueTask
		if err := rows.Scan(&d.ID, &d.Title, &d.DueDate); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
