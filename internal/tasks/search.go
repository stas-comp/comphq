package tasks

import (
	"context"
	"html/template"
	"sort"
	"strconv"
	"time"

	"github.com/stas-comp/comphq/internal/app/wordmatch"
)

// Job search (SPEC B13.4, gates 7.20-7.26, D-94). On Tasks pages the top
// bar's search box finds jobs: every word typed has to appear in the job's
// title or notes, case-insensitively, and the last word may be half-typed
// (a substring match makes that work with no extra machinery).
//
// The matching is done here in Go rather than with SQL LIKE: SQLite's LIKE
// and lower() only fold ASCII, so "électricité" would not find "Électricité".
// Reading every job's title and notes is one indexed pass over a few thousand
// short rows, well inside gate 7.25's second (proved in the speed layer).

const (
	searchLimit      = 20
	minSearchLetters = 2
)

// SearchWords takes words the way the article search does (B4); see wordmatch.
func SearchWords(query string) []string { return wordmatch.Words(query) }

// JobHit is one search result, ready for the panel (as JSON) or the results
// page. Title and Passage are HTML-escaped, with the matched words wrapped in
// <mark>.
type JobHit struct {
	ID       int64         `json:"id"`
	URL      string        `json:"url"`
	Title    template.HTML `json:"title"`
	Passage  template.HTML `json:"passage"`
	Column   string        `json:"column"`
	People   []string      `json:"people"`
	Due      string        `json:"due"`
	Finished bool          `json:"finished"`

	stage      string
	position   int
	titleMatch bool
}

// SearchJobs finds jobs for the words in query (gates 7.21, 7.22). Removed
// jobs are never found; finished ones are, including those tucked away after
// 14 days. Unfinished jobs come before finished ones, a title match before a
// notes-only match, then column order (In progress, To do, Ideas, Done), then
// priority order. At most searchLimit are returned; excludeID (0 for none)
// leaves one job out, for the Linked jobs picker (B13.6).
func (s *Store) SearchJobs(ctx context.Context, query string, excludeID int64, today time.Time) ([]JobHit, error) {
	words := SearchWords(query)
	if len(words) == 0 {
		return []JobHit{}, nil
	}
	lowered := make([][]rune, len(words))
	for i, w := range words {
		lowered[i] = lowerRunes(w)
	}

	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, title, notes, stage, position, due_date, repeat, COALESCE(shows_from, '') FROM tasks WHERE removed_at IS NULL AND id != ?`, excludeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type found struct {
		hit   JobHit
		notes string
		due   string
	}
	var matches []found
	for rows.Next() {
		var id int64
		var title, notes, stage string
		var position int
		var due *string
		var repeat, showsFrom string
		if err := rows.Scan(&id, &title, &notes, &stage, &position, &due, &repeat, &showsFrom); err != nil {
			return nil, err
		}
		titleLow, notesLow := lowerRunes(title), lowerRunes(notes)
		all, inTitle := true, true
		for _, w := range lowered {
			t, n := indexRunes(titleLow, w) >= 0, indexRunes(notesLow, w) >= 0
			if !t && !n {
				all = false
				break
			}
			if !t {
				inTitle = false
			}
		}
		if !all {
			continue
		}
		f := found{notes: notes}
		column := stageLabels[stage]
		// A waiting yearly job is found, marked with when it will appear (gate 8.50):
		// it is in no column people can see yet.
		if repeat == repeatYearly && showsFrom > dateOf(today) {
			if d, err := time.Parse(dateLayout, showsFrom); err == nil {
				column = "Yearly · appears " + d.Format("Mon 2 Jan")
			}
		}
		f.hit = JobHit{
			ID: id, URL: "/tasks/" + strconv.FormatInt(id, 10), Column: column, Finished: stage == StageDone,
			stage: stage, position: position, titleMatch: inTitle,
			Title: highlight(title, lowered),
		}
		if due != nil {
			f.due = *due
		}
		matches = append(matches, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stageRank := map[string]int{StageDoing: 0, StageTodo: 1, StageIdea: 2, StageDone: 3}
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i].hit, matches[j].hit
		if a.Finished != b.Finished {
			return !a.Finished
		}
		if a.titleMatch != b.titleMatch {
			return a.titleMatch
		}
		if stageRank[a.stage] != stageRank[b.stage] {
			return stageRank[a.stage] < stageRank[b.stage]
		}
		return a.position < b.position
	})
	if len(matches) > searchLimit {
		matches = matches[:searchLimit]
	}

	ids := make([]int64, len(matches))
	for i, m := range matches {
		ids[i] = m.hit.ID
	}
	assignees, err := s.assigneesFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	hits := make([]JobHit, 0, len(matches))
	for _, m := range matches {
		h := m.hit
		h.Due = shortDueLabel(m.due, today)
		h.People = []string{}
		for _, a := range assignees[h.ID] {
			h.People = append(h.People, a.Name)
		}
		h.Passage = notesPassage(m.notes, lowered)
		hits = append(hits, h)
	}
	return hits, nil
}

// lowerRunes and indexRunes are the shared word match (internal/app/wordmatch).
func lowerRunes(s string) []rune { return wordmatch.Lower(s) }

func indexRunes(hay, needle []rune) int { return wordmatch.Index(hay, needle) }

// highlight and notesPassage are the shared marking (internal/app/wordmatch).
func highlight(text string, words [][]rune) template.HTML { return wordmatch.Highlight(text, words) }

func notesPassage(notes string, words [][]rune) template.HTML { return wordmatch.Passage(notes, words) }
