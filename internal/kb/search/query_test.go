package search

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// indexWordsRow inserts one kb_search_words row directly, the way
// rewriteSearchRowsFromBlocks writes it alongside kb_search (SPEC B12.5).
func indexWordsRow(t *testing.T, sqlDB *sql.DB, articleID int64, blockID int, title, body string) {
	t.Helper()
	if _, err := sqlDB.Exec(
		`INSERT INTO kb_search_words (title, body, article_id, block_id) VALUES (?, ?, ?, ?)`,
		title, body, articleID, blockID,
	); err != nil {
		t.Fatalf("index words row: %v", err)
	}
}

// SPEC gate 6.30's own five words: a half-typed last word finds the
// whole words it's the start of, even though kb_search's stemmer alone
// would miss every one of them (the fault B12.5 describes).
func TestSearchHalfTypedWordFindsWholeWord(t *testing.T) {
	cases := []struct{ typed, whole string }{
		{"pay", "payment"},
		{"busin", "business"},
		{"generat", "generation"},
		{"voluntee", "volunteer"},
		{"happin", "happiness"},
	}
	for i, c := range cases {
		t.Run(c.typed, func(t *testing.T) {
			sqlDB := openTestDB(t)
			articleID := int64(i + 1)
			body := "A note about " + c.whole + " for the team."
			indexRow(t, sqlDB, articleID, 1, "", body)
			indexWordsRow(t, sqlDB, articleID, 1, "", body)

			results, err := Search(sqlDB, c.typed, 20)
			if err != nil {
				t.Fatalf("Search(%q): %v", c.typed, err)
			}
			if len(results) != 1 || results[0].ArticleID != articleID {
				t.Fatalf("Search(%q) = %+v, want one result for article %d", c.typed, results, articleID)
			}

			highlighted, err := HighlightBlock(sqlDB, articleID, 1, c.typed)
			if err != nil {
				t.Fatalf("HighlightBlock(%q): %v", c.typed, err)
			}
			want := "<mark>" + c.whole + "</mark>"
			if !strings.Contains(highlighted, want) {
				t.Errorf("HighlightBlock(%q) = %q, want it to contain %q (gate 6.32)", c.typed, highlighted, want)
			}
		})
	}
}

// SPEC gate 6.35: a prefix with more than 30 matching whole words takes
// only the 30 most common, most common first.
func TestExpandLastWordCapsAtThirtyMostCommon(t *testing.T) {
	sqlDB := openTestDB(t)
	// 35 distinct words "cat0".."cat34", each its own row so a whole
	// document either contains a word or doesn't: cat0 appears in 35
	// separate rows (most common), cat34 in exactly 1 (least common).
	var rowID int64
	for n := 0; n < 35; n++ {
		word := fmt.Sprintf("cat%d", n)
		occurrences := 35 - n
		for i := 0; i < occurrences; i++ {
			rowID++
			indexWordsRow(t, sqlDB, rowID, 1, "", "the word is "+word)
		}
	}

	got, err := expandLastWord(sqlDB, "cat")
	if err != nil {
		t.Fatalf("expandLastWord: %v", err)
	}
	if len(got) != 30 {
		t.Fatalf("expandLastWord returned %d words, want 30", len(got))
	}
	for i, term := range got {
		want := fmt.Sprintf("cat%d", i)
		if term != want {
			t.Errorf("got[%d] = %q, want %q (most common first)", i, term, want)
		}
	}
	for _, excluded := range []string{"cat30", "cat31", "cat32", "cat33", "cat34"} {
		for _, term := range got {
			if term == excluded {
				t.Errorf("expandLastWord included %q, one of the 5 least common — should have been capped out", excluded)
			}
		}
	}
}

// SPEC gate 6.34: this is what a rollback-and-upgrade actually exercises
// in the container test — kb_search_words rebuilds itself from
// kb_search, so an article published or edited while rolled back to
// v1.2.0 (which writes kb_search but has never heard of
// kb_search_words) is found by a half-typed word the moment the newer
// binary starts again, with no separate migration step involved.
func TestRebuildVocabularyMirrorsKbSearch(t *testing.T) {
	sqlDB := openTestDB(t)
	// Simulates an article published under v1.2.0: kb_search has it,
	// kb_search_words has never heard of it — and specifically a word
	// where the mismatch bites (D-85's own "pay" -> "pai" example), so
	// this proves the rebuild, not just an unstemmed literal prefix that
	// would have matched either way.
	indexRow(t, sqlDB, 1, 0, "Invoices", "")
	indexRow(t, sqlDB, 1, 1, "", "Ask about the payment schedule.")

	var before int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_search_words WHERE article_id = 1`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before != 0 {
		t.Fatalf("kb_search_words rows before rebuild = %d, want 0", before)
	}
	if results, err := Search(sqlDB, "pay", 20); err != nil {
		t.Fatalf("Search before rebuild: %v", err)
	} else if len(results) != 0 {
		t.Fatalf(`Search("pay") before rebuild = %+v, want no results (kb_search's stemmer alone misses "payment", D-85)`, results)
	}

	if err := RebuildVocabulary(sqlDB); err != nil {
		t.Fatalf("RebuildVocabulary: %v", err)
	}

	var after int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_search_words WHERE article_id = 1`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != 2 { // title block (0) + the one body block (1)
		t.Errorf("kb_search_words rows after rebuild = %d, want 2", after)
	}
	if results, err := Search(sqlDB, "pay", 20); err != nil {
		t.Fatalf("Search after rebuild: %v", err)
	} else if len(results) != 1 || results[0].ArticleID != 1 {
		t.Errorf(`Search("pay") after rebuild = %+v, want one result for article 1`, results)
	}
}

// Rebuilding must also be idempotent and safe to run against an
// already-populated kb_search_words (every ordinary start-up, not just
// one following a rollback).
func TestRebuildVocabularyIsIdempotent(t *testing.T) {
	sqlDB := openTestDB(t)
	body := "Ask about the payment schedule."
	indexRow(t, sqlDB, 1, 0, "Invoices", "")
	indexRow(t, sqlDB, 1, 1, "", body)
	indexWordsRow(t, sqlDB, 1, 0, "Invoices", "")
	indexWordsRow(t, sqlDB, 1, 1, "", body)

	for i := 0; i < 2; i++ {
		if err := RebuildVocabulary(sqlDB); err != nil {
			t.Fatalf("RebuildVocabulary (pass %d): %v", i, err)
		}
	}

	var count int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_search_words WHERE article_id = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("kb_search_words rows for article 1 = %d, want 2 (no duplicates across repeated rebuilds)", count)
	}
}

// SPEC gate 6.30: "a prefix matching nothing leaves the query unchanged."
func TestBuildQueryExpandedUnchangedWhenNothingMatches(t *testing.T) {
	sqlDB := openTestDB(t)
	// kb_search_words is empty: no article has ever been published.
	plain, plainOK := BuildQuery("zznomatch")
	expanded, expandedOK, err := buildQueryExpanded(sqlDB, "zznomatch")
	if err != nil {
		t.Fatalf("buildQueryExpanded: %v", err)
	}
	if !plainOK || !expandedOK {
		t.Fatalf("ok = %v, %v, want both true", plainOK, expandedOK)
	}
	if plain != expanded {
		t.Errorf("expanded query = %q, want it unchanged from BuildQuery's %q", expanded, plain)
	}
}
