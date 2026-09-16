// Package search builds FTS5 queries over kb_search and turns raw rows into
// ranked, highlighted results (SPEC B4 "Article blocks and search").
package search

import (
	"regexp"
	"strings"
)

var tokenRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

// BuildQuery turns raw search-box input into an FTS5 MATCH expression:
// letter/digit runs only (everything else, including FTS5's own operator
// syntax, is discarded), each run quoted so words like "AND" or "NEAR"
// never become operators, and the last run is a prefix match. Input with
// fewer than 2 significant characters isn't a search at all: ok is false
// and the caller shouldn't query.
func BuildQuery(raw string) (query string, ok bool) {
	tokens := tokenRe.FindAllString(raw, -1)

	total := 0
	for _, t := range tokens {
		total += len([]rune(t))
	}
	if total < 2 {
		return "", false
	}

	parts := make([]string, len(tokens))
	for i, t := range tokens {
		q := `"` + t + `"`
		if i == len(tokens)-1 {
			q += "*" // FTS5: a quoted string followed by * is a prefix match.
		}
		parts[i] = q
	}
	return strings.Join(parts, " "), true
}
