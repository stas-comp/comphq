// Package wordmatch is the word match every "type a few words" search shares
// (D-94): the query is split into runs of Unicode letters and digits, each
// word is compared case-insensitively and may be half-typed (a substring
// match), and every word has to be found. It is done in Go rather than with
// SQL LIKE, which only folds ASCII, so "électricité" finds "Électricité".
//
// It lives under internal/app so that sections can use it without importing
// each other (SPEC B2).
package wordmatch

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// MinLetters is how many letters a query needs in total before it finds
// anything.
const MinLetters = 2

// Words takes words the way the article search does (B4): runs of Unicode
// letters and digits. A query shorter than two characters in total finds
// nothing, so it returns no words.
func Words(query string) []string {
	var words []string
	total := 0
	for _, w := range strings.FieldsFunc(query, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		words = append(words, w)
		total += utf8.RuneCountInString(w)
	}
	if total < MinLetters {
		return nil
	}
	return words
}

// Lower folds each rune on its own, so the result has exactly as many runes as
// the original and an index in one is an index in the other.
func Lower(s string) []rune {
	r := []rune(s)
	for i := range r {
		r[i] = unicode.ToLower(r[i])
	}
	return r
}

// Index is the first position of needle in hay, or -1.
func Index(hay, needle []rune) int {
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

// Query is a parsed query, ready to test many texts.
type Query struct {
	words [][]rune
}

// Parse splits query into its words. An empty Query (Empty reports it) matches
// nothing.
func Parse(query string) Query {
	var q Query
	for _, w := range Words(query) {
		q.words = append(q.words, Lower(w))
	}
	return q
}

// Empty reports whether the query has no words, which finds nothing.
func (q Query) Empty() bool { return len(q.words) == 0 }

// MatchesAll reports whether every word is found in text.
func (q Query) MatchesAll(text string) bool {
	if q.Empty() {
		return false
	}
	low := Lower(text)
	for _, w := range q.words {
		if Index(low, w) < 0 {
			return false
		}
	}
	return true
}

// StartsWithFirst reports whether text starts with the query's first word, which
// the pickers use to put title-prefix matches first.
func (q Query) StartsWithFirst(text string) bool {
	if q.Empty() {
		return false
	}
	return Index(Lower(text), q.words[0]) == 0
}
