package tasks

import (
	"context"
	"html"
	"html/template"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
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
	passageRunes     = 120
	minSearchLetters = 2
)

// SearchWords takes words the way the article search does (B4): runs of
// Unicode letters and digits. A query shorter than two characters in total
// finds nothing, so it returns no words.
func SearchWords(query string) []string {
	var words []string
	total := 0
	for _, w := range strings.FieldsFunc(query, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		words = append(words, w)
		total += utf8.RuneCountInString(w)
	}
	if total < minSearchLetters {
		return nil
	}
	return words
}

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
		`SELECT id, title, notes, stage, position, due_date FROM tasks WHERE removed_at IS NULL AND id != ?`, excludeID)
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
		if err := rows.Scan(&id, &title, &notes, &stage, &position, &due); err != nil {
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
		f.hit = JobHit{
			ID: id, URL: "/tasks/" + strconv.FormatInt(id, 10), Column: stageLabels[stage], Finished: stage == StageDone,
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

// lowerRunes folds each rune on its own, so the result has exactly as many
// runes as the original and an index in one is an index in the other.
func lowerRunes(s string) []rune {
	r := []rune(s)
	for i := range r {
		r[i] = unicode.ToLower(r[i])
	}
	return r
}

func indexRunes(hay, needle []rune) int {
	if len(needle) == 0 || len(needle) > len(hay) {
		return -1
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		match := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// span is a half-open range of rune indexes.
type span struct{ from, to int }

// matchSpans finds every occurrence of every word in text, merged.
func matchSpans(text []rune, words [][]rune) []span {
	low := make([]rune, len(text))
	for i, r := range text {
		low[i] = unicode.ToLower(r)
	}
	var spans []span
	for _, w := range words {
		for start := 0; start < len(low); {
			i := indexRunes(low[start:], w)
			if i < 0 {
				break
			}
			spans = append(spans, span{start + i, start + i + len(w)})
			start += i + len(w)
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].from < spans[j].from })
	var merged []span
	for _, sp := range spans {
		if n := len(merged); n > 0 && sp.from <= merged[n-1].to {
			if sp.to > merged[n-1].to {
				merged[n-1].to = sp.to
			}
			continue
		}
		merged = append(merged, sp)
	}
	return merged
}

// markRange renders runes [from, to) of text as HTML: escaped first, then the
// matched words wrapped in <mark>. Escaping before marking means a title can
// never inject markup.
func markRange(text []rune, spans []span, from, to int) template.HTML {
	var b strings.Builder
	pos := from
	for _, sp := range spans {
		if sp.to <= from || sp.from >= to {
			continue
		}
		s, e := max(sp.from, from), min(sp.to, to)
		b.WriteString(html.EscapeString(string(text[pos:s])))
		b.WriteString("<mark>")
		b.WriteString(html.EscapeString(string(text[s:e])))
		b.WriteString("</mark>")
		pos = e
	}
	b.WriteString(html.EscapeString(string(text[pos:to])))
	return template.HTML(b.String()) //nolint:gosec // every piece above is html.EscapeString'd
}

func highlight(text string, words [][]rune) template.HTML {
	r := []rune(text)
	return markRange(r, matchSpans(r, words), 0, len(r))
}

// notesPassage is about passageRunes of the notes around the first match, with
// the words marked, or "" when no word is in the notes. Newlines become spaces.
func notesPassage(notes string, words [][]rune) template.HTML {
	if notes == "" {
		return ""
	}
	text := []rune(strings.Join(strings.Fields(notes), " "))
	spans := matchSpans(text, words)
	if len(spans) == 0 {
		return ""
	}
	from := max(0, spans[0].from-passageRunes/3)
	to := min(len(text), from+passageRunes)
	// Start and end on word boundaries, so a passage doesn't open mid-word.
	for from > 0 && from < spans[0].from && text[from-1] != ' ' {
		from++
	}
	body := markRange(text, spans, from, to)
	out := string(body)
	if from > 0 {
		out = "…" + out
	}
	if to < len(text) {
		out += "…"
	}
	return template.HTML(out) //nolint:gosec // built from markRange's escaped output
}
