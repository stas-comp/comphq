package search

import (
	"database/sql"
	"strings"
	"testing"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
)

// openTestDB applies the real kb migrations (the same files the app ships)
// to a fresh in-memory database, so this test also proves the FTS5 schema
// works under modernc.org/sqlite on whichever OS runs it (Windows here,
// Linux in CI).
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	migrations, err := db.LoadMigrations(comphq.Migrations, "kb")
	if err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	if err := db.RunMigrations(sqlDB, "test", migrations); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return sqlDB
}

// indexRow inserts one kb_search row directly, the way the Knowledge Base
// section will once it publishes an article (P1-19). blockID 0 is always
// the title; body blocks are numbered from 1.
func indexRow(t *testing.T, sqlDB *sql.DB, articleID int64, blockID int, title, body string) {
	t.Helper()
	if _, err := sqlDB.Exec(
		`INSERT INTO kb_search (title, body, article_id, block_id) VALUES (?, ?, ?, ?)`,
		title, body, articleID, blockID,
	); err != nil {
		t.Fatalf("index row: %v", err)
	}
}

func TestSearchStemmingAndPrefix(t *testing.T) {
	sqlDB := openTestDB(t)
	indexRow(t, sqlDB, 1, 0, "Printer supplies", "")
	indexRow(t, sqlDB, 1, 1, "", "We keep spare toner cartridges for all printers in the stationery cupboard.")

	for _, q := range []string{"printer", "ton"} {
		results, err := Search(sqlDB, q, 20)
		if err != nil {
			t.Fatalf("Search(%q): %v", q, err)
		}
		if len(results) != 1 || results[0].ArticleID != 1 {
			t.Errorf("Search(%q) = %+v, want one result for article 1", q, results)
		}
	}
}

func TestSearchTitleOutranksBody(t *testing.T) {
	sqlDB := openTestDB(t)
	// Article 1: "toner" only in the body.
	indexRow(t, sqlDB, 1, 0, "Stationery cupboard", "")
	indexRow(t, sqlDB, 1, 1, "", "Ask Sam if you can't find the toner.")
	// Article 2: "toner" in the title.
	indexRow(t, sqlDB, 2, 0, "Toner and printer troubleshooting", "")
	indexRow(t, sqlDB, 2, 1, "", "Start by checking the cable is plugged in.")

	results, err := Search(sqlDB, "toner", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Search = %+v, want 2 results", results)
	}
	if results[0].ArticleID != 2 {
		t.Errorf("results[0].ArticleID = %d, want 2 (title match should outrank a body match)", results[0].ArticleID)
	}
}

func TestSearchSnippetEscapesAndHighlights(t *testing.T) {
	sqlDB := openTestDB(t)
	indexRow(t, sqlDB, 1, 0, "Printer supplies", "")
	indexRow(t, sqlDB, 1, 1, "", "<script>alert(1)</script> toner is in the cupboard.")

	results, err := Search(sqlDB, "toner", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search = %+v, want 1 result", results)
	}
	snippet := results[0].Snippet
	if !strings.Contains(snippet, "<mark>toner</mark>") {
		t.Errorf("snippet %q does not contain <mark>toner</mark>", snippet)
	}
	if strings.Contains(snippet, "<script>") {
		t.Errorf("snippet %q was not HTML-escaped", snippet)
	}
	if !strings.Contains(snippet, "&lt;script&gt;") {
		t.Errorf("snippet %q should contain the escaped <script> text", snippet)
	}
}

func TestSearchSymbolFuzzingNeverErrors(t *testing.T) {
	sqlDB := openTestDB(t)
	indexRow(t, sqlDB, 1, 0, "Printer supplies", "")
	indexRow(t, sqlDB, 1, 1, "", "toner cartridges")

	inputs := []string{
		``, `a`, ` `, `"`, `*`, `(`, `-`, `:`, `NEAR`, `AND`, `OR`, `^`, `'`,
		`" * ( - : NEAR AND OR ^ '`,
	}
	for _, in := range inputs {
		if _, err := Search(sqlDB, in, 20); err != nil {
			t.Errorf("Search(%q) returned an error: %v", in, err)
		}
		if _, err := HighlightBlock(sqlDB, 1, 1, in); err != nil {
			t.Errorf("HighlightBlock(%q) returned an error: %v", in, err)
		}
	}
}

func TestSearchExcludesDeletedArticle(t *testing.T) {
	sqlDB := openTestDB(t)
	indexRow(t, sqlDB, 1, 0, "Printer supplies", "")
	indexRow(t, sqlDB, 1, 1, "", "toner cartridges")

	results, err := Search(sqlDB, "toner", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search before delete = %+v, want 1 result", results)
	}

	if _, err := sqlDB.Exec(`DELETE FROM kb_search WHERE article_id = ?`, int64(1)); err != nil {
		t.Fatalf("delete: %v", err)
	}

	results, err = Search(sqlDB, "toner", 20)
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search after delete = %+v, want no results", results)
	}
}

func TestHighlightBlockWrapsMatches(t *testing.T) {
	sqlDB := openTestDB(t)
	indexRow(t, sqlDB, 1, 0, "Printer supplies", "")
	indexRow(t, sqlDB, 1, 1, "", "Ask Sam about the toner cartridges.")

	got, err := HighlightBlock(sqlDB, 1, 1, "toner")
	if err != nil {
		t.Fatalf("HighlightBlock: %v", err)
	}
	if !strings.Contains(got, "<mark>toner</mark>") {
		t.Errorf("HighlightBlock = %q, want it to contain <mark>toner</mark>", got)
	}
}
