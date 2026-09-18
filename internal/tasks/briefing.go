package tasks

import (
	"context"
	"time"
)

// BriefingTask is one row of BriefingTasks' result (SPEC A8): the
// Briefing's own small query interface onto this package, injected at
// registration rather than imported directly (SPEC B2, D-53).
type BriefingTask struct {
	ID        int64
	Title     string
	Notes     string
	DueDate   string
	Position  int
	Assignees []Assignee
}

// BriefingTasks returns every unfinished, unremoved task with a due
// date on or before dueOnOrBefore — overdue ones included (SPEC A8:
// "every unfinished task due on or before the Friday after that
// Saturday"), ordered by due date then board position.
func (s *Store) BriefingTasks(ctx context.Context, dueOnOrBefore time.Time) ([]BriefingTask, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, title, notes, due_date, position
		FROM tasks
		WHERE removed_at IS NULL
		  AND stage != ?
		  AND due_date IS NOT NULL AND due_date != ''
		  AND due_date <= ?
		ORDER BY due_date, position, id
	`, StageDone, dueOnOrBefore.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	// Closed before assigneesFor runs: it issues its own query, and a
	// second query while these rows are still open deadlocks the single
	// connection (D-44).
	var out []BriefingTask
	var ids []int64
	for rows.Next() {
		var t BriefingTask
		if err := rows.Scan(&t.ID, &t.Title, &t.Notes, &t.DueDate, &t.Position); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	assignees, err := s.assigneesFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Assignees = assignees[out[i].ID]
	}
	return out, nil
}
