// Package search builds FTS5 queries over kb_search and turns raw rows into
// ranked, highlighted results (SPEC B4 "Article blocks and search").
package search

import (
	"database/sql"
	"regexp"
	"strings"
)

var tokenRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

// maxExpansions caps how many whole words a half-typed last word can
// expand into (SPEC gate 6.35: a two-letter prefix can't build a huge
// query), most common first (kb_search_words_vocab.cnt).
const maxExpansions = 30

func tokenize(raw string) (tokens []string, ok bool) {
	tokens = tokenRe.FindAllString(raw, -1)
	total := 0
	for _, t := range tokens {
		total += len([]rune(t))
	}
	return tokens, total >= 2
}

func buildFromTokens(tokens []string, lastWordAlternatives []string) string {
	parts := make([]string, len(tokens))
	for i, t := range tokens {
		if i != len(tokens)-1 {
			parts[i] = `"` + t + `"`
			continue
		}
		// FTS5: a quoted string followed by * is a prefix match.
		last := `"` + t + `"*`
		if len(lastWordAlternatives) > 0 {
			alts := make([]string, 0, len(lastWordAlternatives)+1)
			alts = append(alts, last)
			for _, w := range lastWordAlternatives {
				alts = append(alts, `"`+w+`"`)
			}
			last = "(" + strings.Join(alts, " OR ") + ")"
		}
		parts[i] = last
	}
	return strings.Join(parts, " ")
}

// BuildQuery turns raw search-box input into an FTS5 MATCH expression:
// letter/digit runs only (everything else, including FTS5's own operator
// syntax, is discarded), each run quoted so words like "AND" or "NEAR"
// never become operators, and the last run is a prefix match. Input with
// fewer than 2 significant characters isn't a search at all: ok is false
// and the caller shouldn't query.
func BuildQuery(raw string) (query string, ok bool) {
	tokens, ok := tokenize(raw)
	if !ok {
		return "", false
	}
	return buildFromTokens(tokens, nil), true
}

// buildQueryExpanded is BuildQuery plus gate 6.30's half-typed-word
// lookup (SPEC B12.5, D-85). kb_search's porter stemmer can turn a
// half-typed prefix into something that isn't a prefix of the stemmed
// whole word ("pay" -> "pai", not a prefix of "payment"), so the last
// word is also matched against every whole word in the knowledge base's
// own vocabulary (kb_search_words, unstemmed) that it's a prefix of.
// Finished words are untouched: BuildQuery's own stemmed prefix match
// stays in the OR alongside them, and a prefix matching nothing leaves
// the query exactly as BuildQuery would have built it (gate 6.30's "a
// prefix matching nothing leaves the query unchanged").
func buildQueryExpanded(db *sql.DB, raw string) (query string, ok bool, err error) {
	tokens, ok := tokenize(raw)
	if !ok {
		return "", false, nil
	}
	alternatives, err := expandLastWord(db, tokens[len(tokens)-1])
	if err != nil {
		return "", false, err
	}
	return buildFromTokens(tokens, alternatives), true, nil
}

// expandLastWord looks up every whole word in the vocabulary that word is
// a prefix of, most common first, capped at maxExpansions. word is
// lower-cased first: kb_search_words is tokenized with unicode61, which
// folds case, so terms are stored lower-case and a raw-cased comparison
// would silently match nothing.
func expandLastWord(db *sql.DB, word string) ([]string, error) {
	word = strings.ToLower(word)
	rows, err := db.Query(`
		SELECT term FROM kb_search_words_vocab
		WHERE term >= ? AND term < ? || char(0x10FFFF)
		ORDER BY cnt DESC
		LIMIT ?
	`, word, word, maxExpansions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terms []string
	for rows.Next() {
		var term string
		if err := rows.Scan(&term); err != nil {
			return nil, err
		}
		terms = append(terms, term)
	}
	return terms, rows.Err()
}
