package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
)

// sizeLabels are the full words shown for a task's size (SPEC gate
// 2.26: cards and the Team view show Small/Medium/Large, not S/M/L).
var sizeLabels = map[string]string{
	"S": "Small",
	"M": "Medium",
	"L": "Large",
}

// UpdateInput is every field the task details page can change (SPEC
// gate 2.06's "editing all its details"; gate 2.26's size).
type UpdateInput struct {
	Title     string
	Notes     string
	Size      string
	Stage     string
	DueDate   string // "" clears it
	PersonIDs []int64
}

// Update applies every change from the details page in one transaction,
// recording one activity row per kind of change actually made rather
// than a single generic "edited" row for everything — gate 2.06 and
// 2.26 both require Activity to name what changed (moved, assigned,
// unassigned, due date, size). A stage change here has no drop position
// to honour, so the task goes to the bottom of its new stage, the same
// default the "Move to…" button (P2-02) already uses.
func (s *Store) Update(ctx context.Context, taskID int64, input UpdateInput, actorID int64, now time.Time) error {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return ErrEmptyTitle
	}
	if !validSize(input.Size) {
		return ErrInvalidSize
	}
	if !validStage(input.Stage) {
		return ErrInvalidStage
	}
	if input.DueDate != "" && !format.ValidISODate(input.DueDate) {
		return ErrInvalidDate
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldTitle, oldNotes, oldSize, oldStage string
	var oldDueDate sql.NullString
	err = tx.QueryRowContext(ctx,
		`SELECT title, notes, size, stage, due_date FROM tasks WHERE id = ? AND removed_at IS NULL`, taskID,
	).Scan(&oldTitle, &oldNotes, &oldSize, &oldStage, &oldDueDate)
	if err == sql.ErrNoRows {
		return ErrTaskNotFound
	}
	if err != nil {
		return err
	}

	oldPersonIDs, err := assigneeIDs(ctx, tx, taskID)
	if err != nil {
		return err
	}

	nowStr := now.UTC().Format(time.RFC3339)

	if title != oldTitle || input.Notes != oldNotes {
		if err := recordActivity(ctx, tx, taskID, actorID, "edited", nil, nowStr); err != nil {
			return err
		}
	}

	if input.DueDate != oldDueDate.String {
		if err := recordActivity(ctx, tx, taskID, actorID, "due_changed", dueChangedDetail{DueDate: input.DueDate}, nowStr); err != nil {
			return err
		}
	}

	if input.Size != oldSize {
		if err := recordActivity(ctx, tx, taskID, actorID, "size_changed", sizeChangedDetail{Size: input.Size}, nowStr); err != nil {
			return err
		}
	}

	removedIDs, addedIDs := diffPersonIDs(oldPersonIDs, input.PersonIDs)
	for _, personID := range removedIDs {
		if _, err := tx.ExecContext(ctx, `DELETE FROM task_assignees WHERE task_id = ? AND person_id = ?`, taskID, personID); err != nil {
			return err
		}
		if err := recordActivity(ctx, tx, taskID, actorID, "unassigned", personDetail{PersonID: personID}, nowStr); err != nil {
			return err
		}
	}
	for _, personID := range addedIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_assignees (task_id, person_id) VALUES (?, ?)`, taskID, personID); err != nil {
			return err
		}
		if err := recordActivity(ctx, tx, taskID, actorID, "assigned", personDetail{PersonID: personID}, nowStr); err != nil {
			return err
		}
	}

	if input.Stage != oldStage {
		newOrder, err := stageOrder(ctx, tx, input.Stage, taskID)
		if err != nil {
			return err
		}
		newOrder = append(newOrder, taskID)
		if err := renumber(ctx, tx, newOrder); err != nil {
			return err
		}
		oldOrder, err := stageOrder(ctx, tx, oldStage, taskID)
		if err != nil {
			return err
		}
		if err := renumber(ctx, tx, oldOrder); err != nil {
			return err
		}
		if err := recordActivity(ctx, tx, taskID, actorID, "moved", movedDetail{From: oldStage, To: input.Stage}, nowStr); err != nil {
			return err
		}
	}

	var dueDate any
	if input.DueDate != "" {
		dueDate = input.DueDate
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET title = ?, notes = ?, size = ?, stage = ?, due_date = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		title, input.Notes, input.Size, input.Stage, dueDate, actorID, nowStr, taskID,
	); err != nil {
		return err
	}
	if err := updateDoneAtForStageChange(ctx, tx, taskID, oldStage, input.Stage, nowStr); err != nil {
		return err
	}

	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func assigneeIDs(ctx context.Context, tx *sql.Tx, taskID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT person_id FROM task_assignees WHERE task_id = ?`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// diffPersonIDs reports which of oldIDs no longer appear in newIDs
// (removed) and which of newIDs weren't already in oldIDs (added).
func diffPersonIDs(oldIDs, newIDs []int64) (removed, added []int64) {
	oldSet := make(map[int64]bool, len(oldIDs))
	for _, id := range oldIDs {
		oldSet[id] = true
	}
	newSet := make(map[int64]bool, len(newIDs))
	for _, id := range newIDs {
		newSet[id] = true
	}
	for _, id := range oldIDs {
		if !newSet[id] {
			removed = append(removed, id)
		}
	}
	for _, id := range newIDs {
		if !oldSet[id] {
			added = append(added, id)
		}
	}
	return removed, added
}

// Detail payloads for task_activity.detail (SPEC B3: "short JSON").
type dueChangedDetail struct {
	DueDate string `json:"due_date"`
}
type sizeChangedDetail struct {
	Size string `json:"size"`
}
type personDetail struct {
	PersonID int64 `json:"person_id"`
}
type movedDetail struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// recordActivity inserts one task_activity row. A nil detail (e.g.
// "created", "edited") is stored as "{}", matching the empty-object
// convention Create already uses, rather than JSON's "null".
func recordActivity(ctx context.Context, tx *sql.Tx, taskID, actorID int64, action string, detail any, at string) error {
	detailJSON := "{}"
	if detail != nil {
		b, err := json.Marshal(detail)
		if err != nil {
			return err
		}
		detailJSON = string(b)
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO task_activity (task_id, person_id, action, detail, at) VALUES (?, ?, ?, ?, ?)`,
		taskID, actorID, action, detailJSON, at,
	)
	return err
}

// Activity is one task_activity row, rendered as a single sentence plus
// a pre-formatted date (SPEC gate 2.06: "Sam moved this to In progress
// · Sat 19 Sep"). Names resolve by id, live — a rename shows
// immediately, the same as KB History (SPEC B4); people are never
// hard-deleted, so the join always finds a name.
type Activity struct {
	Text string
	At   string
}

// ListActivity returns a task's activity, newest first (matching KB
// History's own convention).
func (s *Store) ListActivity(ctx context.Context, taskID int64) ([]Activity, error) {
	type rawActivity struct{ action, detail, at, actorName string }

	// Read every row and close the cursor before rendering any of them:
	// rendering "assigned"/"unassigned" queries s.DB again for the
	// target's name, and the app's single SQLite connection
	// (SetMaxOpenConns(1)) can't serve that second query while this
	// one's rows are still open — it would deadlock.
	var raw []rawActivity
	err := func() error {
		rows, err := s.DB.QueryContext(ctx, `
			SELECT a.action, a.detail, a.at, p.name
			FROM task_activity a
			JOIN people p ON p.id = a.person_id
			WHERE a.task_id = ?
			ORDER BY a.id DESC
		`, taskID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r rawActivity
			if err := rows.Scan(&r.action, &r.detail, &r.at, &r.actorName); err != nil {
				return err
			}
			raw = append(raw, r)
		}
		return rows.Err()
	}()
	if err != nil {
		return nil, err
	}

	out := make([]Activity, 0, len(raw))
	for _, r := range raw {
		parsedAt, err := time.Parse(time.RFC3339, r.at)
		if err != nil {
			return nil, err
		}
		text, err := s.activityText(ctx, r.action, r.detail, r.actorName)
		if err != nil {
			return nil, err
		}
		out = append(out, Activity{Text: text, At: format.Date(parsedAt)})
	}
	return out, nil
}

// activityText renders one row as a single sentence (SPEC gate 2.06);
// the caller appends the pre-formatted date separately.
func (s *Store) activityText(ctx context.Context, action, detail, actorName string) (string, error) {
	switch action {
	case "created":
		return actorName + " created this", nil
	case "edited":
		return actorName + " edited this", nil
	case "reopened":
		return actorName + " reopened this", nil
	case "removed":
		return actorName + " removed this", nil
	case "restored":
		return actorName + " restored this", nil
	case "moved":
		var d movedDetail
		if err := json.Unmarshal([]byte(detail), &d); err != nil {
			return "", err
		}
		return actorName + " moved this to " + stageLabels[d.To], nil
	case "due_changed":
		var d dueChangedDetail
		if err := json.Unmarshal([]byte(detail), &d); err != nil {
			return "", err
		}
		if d.DueDate == "" {
			return actorName + " removed the due date", nil
		}
		due, err := time.Parse("2006-01-02", d.DueDate)
		if err != nil {
			return "", err
		}
		return actorName + " changed the due date to " + format.Date(due), nil
	case "size_changed":
		var d sizeChangedDetail
		if err := json.Unmarshal([]byte(detail), &d); err != nil {
			return "", err
		}
		return actorName + " changed the size to " + sizeLabels[d.Size], nil
	case "step_added", "step_removed", "step_restored":
		var d stepDetail
		if err := json.Unmarshal([]byte(detail), &d); err != nil {
			return "", err
		}
		verb := map[string]string{"step_added": "added", "step_removed": "removed", "step_restored": "restored"}[action]
		return actorName + " " + verb + " the step “" + d.Text + "”", nil
	case "step_renamed":
		var d stepDetail
		if err := json.Unmarshal([]byte(detail), &d); err != nil {
			return "", err
		}
		return actorName + " renamed the step “" + d.From + "” to “" + d.Text + "”", nil
	case "assigned", "unassigned":
		var d personDetail
		if err := json.Unmarshal([]byte(detail), &d); err != nil {
			return "", err
		}
		var targetName string
		if err := s.DB.QueryRowContext(ctx, `SELECT name FROM people WHERE id = ?`, d.PersonID).Scan(&targetName); err != nil {
			return "", err
		}
		return actorName + " " + action + " " + targetName, nil
	default:
		return actorName + " " + action, nil
	}
}
