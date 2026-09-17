package search

import (
	"database/sql"
	"html"
	"strings"
)

// Private-use Unicode characters used as snippet()/highlight() markers.
// They can't appear in real content, and aren't touched by HTML-escaping,
// so escape-then-replace is always safe (SPEC B4).
const (
	markStart = ""
	markEnd   = ""
)

// Result is one article's best-matching block, ready to render.
type Result struct {
	ArticleID int64
	BlockID   int
	Snippet   string // HTML: escaped, with <mark>…</mark> around matches
}

// Search runs raw against kb_search and returns the top `limit` articles,
// each represented by its single best-ranked block (SPEC B4: "Take the
// best-ranked block per article, then the top 20 articles"). Input that
// BuildQuery rejects (too short) returns no results and no error, matching
// gate 1.32 ("empty or 1-character input never causes an error").
func Search(db *sql.DB, raw string, limit int) ([]Result, error) {
	query, ok := BuildQuery(raw)
	if !ok {
		return nil, nil
	}

	rows, err := db.Query(`
		WITH ranked AS (
			SELECT
				article_id,
				block_id,
				bm25(kb_search, 10.0, 1.0) AS rank,
				snippet(kb_search, -1, ?, ?, '…', 12) AS snippet
			FROM kb_search
			WHERE kb_search MATCH ?
		),
		best AS (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY article_id ORDER BY rank) AS rn
			FROM ranked
		)
		SELECT article_id, block_id, snippet
		FROM best
		WHERE rn = 1
		ORDER BY rank
		LIMIT ?
	`, markStart, markEnd, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		var snippet string
		if err := rows.Scan(&r.ArticleID, &r.BlockID, &snippet); err != nil {
			return nil, err
		}
		r.Snippet = markersToHTML(snippet)
		results = append(results, r)
	}
	return results, rows.Err()
}

// HighlightBlock returns the full text of one block (title block is 0, per
// SPEC B3) with every match of raw wrapped in <mark>, HTML-escaped
// elsewhere. Used on the article page to highlight the block a search
// result linked to (SPEC B4: "Matched words come from the server via
// highlight() on that block").
func HighlightBlock(db *sql.DB, articleID int64, blockID int, raw string) (string, error) {
	query, ok := BuildQuery(raw)
	if !ok {
		return "", nil
	}

	// Each row populates only one of the two columns (SPEC B3: block 0 is
	// the title, 1..n are body blocks), and unlike snippet(), highlight()
	// has no "auto-pick the column" mode — so pick the column ourselves.
	col := 1 // body
	if blockID == 0 {
		col = 0 // title
	}

	var highlighted string
	err := db.QueryRow(`
		SELECT highlight(kb_search, ?, ?, ?)
		FROM kb_search
		WHERE kb_search MATCH ? AND article_id = ? AND block_id = ?
	`, col, markStart, markEnd, query, articleID, blockID).Scan(&highlighted)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return markersToHTML(highlighted), nil
}

// HighlightMatches returns every block of one article that matches raw —
// including block 0, the title, since a title match can rank as the best
// block a search result links to — each with its matched words wrapped in
// <mark> (SPEC B4: the article page marks every block a query hits, not
// just the one a result link scrolls to; the client picks which one to
// scroll to from the URL's own #b-N fragment, which never reaches the
// server). Block 0 needs its own query: unlike snippet(), highlight() has
// no auto-pick-the-column mode, and the title lives in a different column
// than body blocks (SPEC B3).
func HighlightMatches(db *sql.DB, articleID int64, raw string) (map[int]string, error) {
	query, ok := BuildQuery(raw)
	if !ok {
		return nil, nil
	}

	out := make(map[int]string)

	var titleHighlight sql.NullString
	err := db.QueryRow(`
		SELECT highlight(kb_search, 0, ?, ?)
		FROM kb_search
		WHERE kb_search MATCH ? AND article_id = ? AND block_id = 0
	`, markStart, markEnd, query, articleID).Scan(&titleHighlight)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if titleHighlight.Valid {
		out[0] = markersToHTML(titleHighlight.String)
	}

	rows, err := db.Query(`
		SELECT block_id, highlight(kb_search, 1, ?, ?)
		FROM kb_search
		WHERE kb_search MATCH ? AND article_id = ? AND block_id != 0
	`, markStart, markEnd, query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var blockID int
		var highlighted string
		if err := rows.Scan(&blockID, &highlighted); err != nil {
			return nil, err
		}
		out[blockID] = markersToHTML(highlighted)
	}
	return out, rows.Err()
}

func markersToHTML(s string) string {
	escaped := html.EscapeString(s)
	escaped = strings.ReplaceAll(escaped, markStart, "<mark>")
	escaped = strings.ReplaceAll(escaped, markEnd, "</mark>")
	return escaped
}
