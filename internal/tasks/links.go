package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Linked jobs (SPEC B13.6, D-96, gates 7.40-7.49). A link joins two jobs in
// one of three ways, seen from the job being looked at:
//
//	first   the other job must be finished before this one (stored as a
//	        "before" link from the other job to this one)
//	then    this job must be finished before the other (a "before" link from
//	        this one to the other)
//	related they belong together, in any order (stored once, lower id first)
//
// A job with an unfinished "first" job is WAITING. That is only a label (D-96):
// nothing is ever blocked, because anyone can do anything (A3).

// The kinds a caller asks for.
const (
	LinkFirst   = "first"
	LinkThen    = "then"
	LinkRelated = "related"
)

const (
	linkBefore  = "before"
	maxTaskLinks = 20
)

// Refusals (gate 7.45). Each is a plain message that handlers show as it is;
// nothing changes when one is returned.
var (
	ErrLinkSelf         = errors.New("A job can't be linked to itself.")
	ErrLinkExists       = errors.New("Those two jobs are already linked. Remove that link first to change it.")
	ErrLinkTooMany      = errors.New("A job can have at most 20 links.")
	ErrLinkOtherRemoved = errors.New("That job has just been removed, so it can't be linked.")
	ErrLinkKind         = errors.New("Say whether it comes first, comes after, or is related.")
	ErrLinkNotFound     = errors.New("That link isn't there any more.")
)

// LinkLoopError is a "do first" link that would go round in a circle: the other
// job (directly or through others) already has to wait for this one, or this
// one already has to wait for the other.
type LinkLoopError struct {
	OtherTitle string
	Kind       string
}

func (e *LinkLoopError) Error() string {
	if e.Kind == LinkFirst {
		return fmt.Sprintf("“%s” already has to wait for this job, so it can't also come first.", e.OtherTitle)
	}
	return fmt.Sprintf("This job already has to wait for “%s”, so it can't also come first.", e.OtherTitle)
}

// linkDetail is what a "linked" or "unlinked" History row remembers. The other
// job's title is kept so that a later rename can't rewrite history (as the
// step_* rows do), and Kind is how the job whose History this is sees it.
type linkDetail struct {
	OtherID    int64  `json:"other_id"`
	OtherTitle string `json:"other_title"`
	Kind       string `json:"kind"`
}

func flipKind(kind string) string {
	switch kind {
	case LinkFirst:
		return LinkThen
	case LinkThen:
		return LinkFirst
	}
	return kind
}

// LinkView is one line of a job's Linked jobs: the other job, which way the
// link points, and how to remove it (gates 7.41, 7.44).
type LinkView struct {
	LinkID   int64
	OtherID  int64
	Title    string
	Column   string
	Finished bool
	Removed  bool
}

// TaskLinks are a job's live links in the three groups of gate 7.41. Links
// whose other job is removed are not here: they come back when it is restored
// (gate 7.46).
type TaskLinks struct {
	DoFirst []LinkView // jobs this one is waiting on
	Then    []LinkView // jobs waiting on this one
	Related []LinkView
}

// Count is how many links the job has.
func (l TaskLinks) Count() int { return len(l.DoFirst) + len(l.Then) + len(l.Related) }

// ListLinks reads a job's links with one query, joined to the other job's title
// and column. Finished jobs keep their links and show Finished (gate 7.46).
func (s *Store) ListLinks(ctx context.Context, taskID int64) (TaskLinks, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT l.id, l.kind, l.task_id, o.id, o.title, o.stage
		FROM task_links l
		JOIN tasks o ON o.id = CASE WHEN l.task_id = ?1 THEN l.other_task_id ELSE l.task_id END
		WHERE (l.task_id = ?1 OR l.other_task_id = ?1)
		  AND l.removed_at IS NULL AND o.removed_at IS NULL
		ORDER BY l.id`, taskID)
	if err != nil {
		return TaskLinks{}, err
	}
	defer rows.Close()
	var out TaskLinks
	for rows.Next() {
		var linkID, from, otherID int64
		var kind, title, stage string
		if err := rows.Scan(&linkID, &kind, &from, &otherID, &title, &stage); err != nil {
			return TaskLinks{}, err
		}
		v := LinkView{LinkID: linkID, OtherID: otherID, Title: title, Column: stageLabels[stage], Finished: stage == StageDone}
		switch {
		case kind == "related":
			out.Related = append(out.Related, v)
		case from == taskID: // this job comes before the other
			out.Then = append(out.Then, v)
		default: // the other job comes before this one
			out.DoFirst = append(out.DoFirst, v)
		}
	}
	return out, rows.Err()
}

// AddLink links a job to another (gates 7.40, 7.45). All the checks are inside
// one write transaction, so two people linking at once can't both get past
// the same rule.
func (s *Store) AddLink(ctx context.Context, taskID, otherID int64, kind string, actorID int64, now time.Time) error {
	if kind != LinkFirst && kind != LinkThen && kind != LinkRelated {
		return ErrLinkKind
	}
	if taskID == otherID {
		return ErrLinkSelf
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var thisTitle, otherTitle string
	var thisRemoved, otherRemoved sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT title, removed_at FROM tasks WHERE id = ?`, taskID).Scan(&thisTitle, &thisRemoved); err != nil {
		if err == sql.ErrNoRows {
			return ErrTaskNotFound
		}
		return err
	}
	if thisRemoved.Valid {
		return ErrTaskNotFound
	}
	if err := tx.QueryRowContext(ctx, `SELECT title, removed_at FROM tasks WHERE id = ?`, otherID).Scan(&otherTitle, &otherRemoved); err != nil {
		if err == sql.ErrNoRows {
			return ErrLinkOtherRemoved
		}
		return err
	}
	if otherRemoved.Valid {
		return ErrLinkOtherRemoved
	}

	// At most one live link between two jobs, of any kind.
	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM task_links WHERE removed_at IS NULL
		   AND ((task_id = ?1 AND other_task_id = ?2) OR (task_id = ?2 AND other_task_id = ?1))`,
		taskID, otherID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrLinkExists
	}

	// At most 20 links on either job (counting only those that show).
	for _, id := range []int64{taskID, otherID} {
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM task_links l
			JOIN tasks o ON o.id = CASE WHEN l.task_id = ?1 THEN l.other_task_id ELSE l.task_id END
			WHERE (l.task_id = ?1 OR l.other_task_id = ?1) AND l.removed_at IS NULL AND o.removed_at IS NULL`, id).Scan(&n); err != nil {
			return err
		}
		if n >= maxTaskLinks {
			return ErrLinkTooMany
		}
	}

	// Which job comes first, for the "before" kinds.
	var from, to int64
	storedKind := linkBefore
	switch kind {
	case LinkFirst:
		from, to = otherID, taskID
	case LinkThen:
		from, to = taskID, otherID
	default:
		storedKind = "related"
		from, to = min(taskID, otherID), max(taskID, otherID)
	}
	if storedKind == linkBefore {
		// A loop: something already has to come before `from` that `to` leads
		// back to, so adding "from before to" would close the circle. Walk the
		// live "before" links forward from `to`, ignoring removed jobs.
		loop, err := reaches(ctx, tx, to, from)
		if err != nil {
			return err
		}
		if loop {
			return &LinkLoopError{OtherTitle: otherTitle, Kind: kind}
		}
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO task_links (task_id, other_task_id, kind, created_by, created_at) VALUES (?, ?, ?, ?, ?)`,
		from, to, storedKind, actorID, nowStr); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "linked", linkDetail{OtherID: otherID, OtherTitle: otherTitle, Kind: kind}, nowStr); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, otherID, actorID, "linked", linkDetail{OtherID: taskID, OtherTitle: thisTitle, Kind: flipKind(kind)}, nowStr); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// reaches reports whether `target` can be reached from `start` by following
// live "before" links forward (a job to the jobs that wait on it), skipping
// removed jobs.
func reaches(ctx context.Context, tx *sql.Tx, start, target int64) (bool, error) {
	seen := map[int64]bool{start: true}
	queue := []int64{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == target {
			return true, nil
		}
		rows, err := tx.QueryContext(ctx, `
			SELECT l.other_task_id FROM task_links l
			JOIN tasks o ON o.id = l.other_task_id
			WHERE l.task_id = ? AND l.kind = 'before' AND l.removed_at IS NULL AND o.removed_at IS NULL`, cur)
		if err != nil {
			return false, err
		}
		var next []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return false, err
			}
			next = append(next, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return false, err
		}
		for _, id := range next {
			if !seen[id] {
				seen[id] = true
				queue = append(queue, id)
			}
		}
	}
	return false, nil
}

// RemoveLink removes a link from both jobs (gate 7.44). It never deletes:
// removed_at and removed_by are set (A3). The link must involve taskID; one
// somebody else already removed is simply gone (ErrLinkNotFound is for a link
// that was never this job's).
func (s *Store) RemoveLink(ctx context.Context, taskID, linkID, actorID int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var from, to int64
	var kind string
	var removed sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT task_id, other_task_id, kind, removed_at FROM task_links WHERE id = ?`, linkID).Scan(&from, &to, &kind, &removed); err != nil {
		if err == sql.ErrNoRows {
			return ErrLinkNotFound
		}
		return err
	}
	if from != taskID && to != taskID {
		return ErrLinkNotFound
	}
	if removed.Valid {
		return nil // somebody else got there first
	}
	otherID := to
	if to == taskID {
		otherID = from
	}
	// How each job sees the link, for the History rows.
	seen := LinkRelated
	if kind == linkBefore {
		seen = LinkThen // this job comes before the other
		if to == taskID {
			seen = LinkFirst
		}
	}
	var thisTitle, otherTitle string
	if err := tx.QueryRowContext(ctx, `SELECT title FROM tasks WHERE id = ?`, taskID).Scan(&thisTitle); err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, `SELECT title FROM tasks WHERE id = ?`, otherID).Scan(&otherTitle); err != nil {
		return err
	}

	nowStr := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `UPDATE task_links SET removed_by = ?, removed_at = ? WHERE id = ?`, actorID, nowStr, linkID); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, taskID, actorID, "unlinked", linkDetail{OtherID: otherID, OtherTitle: otherTitle, Kind: seen}, nowStr); err != nil {
		return err
	}
	if err := recordActivity(ctx, tx, otherID, actorID, "unlinked", linkDetail{OtherID: taskID, OtherTitle: thisTitle, Kind: flipKind(seen)}, nowStr); err != nil {
		return err
	}
	if err := bumpTasksVersion(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// WaitingOn says, for each of the given jobs, which unfinished jobs it is
// waiting on (gate 7.42): the jobs with a live "before" link to it that are
// neither removed nor Done. It is one query for a whole list, so the WAITING
// stamp costs a card nothing; a job that isn't waiting has no entry.
func (s *Store) WaitingOn(ctx context.Context, taskIDs []int64) (map[int64][]string, error) {
	out := make(map[int64][]string)
	if len(taskIDs) == 0 {
		return out, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(taskIDs)), ",")
	args := make([]any, len(taskIDs))
	for i, id := range taskIDs {
		args[i] = id
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT l.other_task_id, o.title
		FROM task_links l
		JOIN tasks o ON o.id = l.task_id
		WHERE l.other_task_id IN (`+placeholders+`)
		  AND l.kind = 'before' AND l.removed_at IS NULL
		  AND o.removed_at IS NULL AND o.stage != 'done'
		ORDER BY l.other_task_id, l.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, err
		}
		out[id] = append(out[id], title)
	}
	return out, rows.Err()
}

// LinksCSV is the export's links.csv (gate 7.48): the job, the linked job,
// what the link means, who linked them and when, and whether it was removed.
type LinksCSV struct{ DB *sql.DB }

// ExportRows returns a header row in plain English, then one row per link in
// the order they were made, removed ones included. "Link" says "do first" when
// the job in the first column has to be finished before the linked job in the
// second, or "related".
func (c LinksCSV) ExportRows(ctx context.Context) ([][]string, error) {
	rows, err := c.DB.QueryContext(ctx, `
		SELECT a.title, b.title, l.kind, p.name, l.created_at, l.removed_at
		FROM task_links l
		JOIN tasks a ON a.id = l.task_id
		JOIN tasks b ON b.id = l.other_task_id
		JOIN people p ON p.id = l.created_by
		ORDER BY l.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := [][]string{{"Job", "Linked job", "Link", "Linked by", "Linked", "Removed"}}
	for rows.Next() {
		var a, b, kind, who, created string
		var removed *string
		if err := rows.Scan(&a, &b, &kind, &who, &created, &removed); err != nil {
			return nil, err
		}
		meaning := "related"
		if kind == linkBefore {
			meaning = "do first"
		}
		out = append(out, []string{a, b, meaning, who, exportDateTime(created), exportDateTimePtr(removed)})
	}
	return out, rows.Err()
}
