package kb

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stas-comp/comphq/internal/people"
)

func testPerson(t *testing.T, sqlDB *sql.DB) int64 {
	t.Helper()
	p, err := (&people.Store{DB: sqlDB}).Create("Sam")
	if err != nil {
		t.Fatalf("create person: %v", err)
	}
	return p.ID
}

func TestPublishNewArticleCreatesVersion1(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)

	store := &ArticleStore{DB: sqlDB}
	article, err := store.Publish(context.Background(), ArticleInput{
		CategoryID: cat.ID,
		Title:      "Changing the toner",
		BodyHTML:   "<p>Open the front cover and pull the cartridge out.</p>",
	}, personID)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if article.VersionNo != 1 {
		t.Errorf("VersionNo = %d, want 1", article.VersionNo)
	}

	var versionCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_article_versions WHERE article_id = ?`, article.ID).Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if versionCount != 1 {
		t.Errorf("version rows = %d, want 1", versionCount)
	}
	var action string
	if err := sqlDB.QueryRow(`SELECT action FROM kb_article_versions WHERE article_id = ?`, article.ID).Scan(&action); err != nil {
		t.Fatal(err)
	}
	if action != "created" {
		t.Errorf("version action = %q, want created", action)
	}

	// Block 0 (title) plus one body block (the <p>).
	var searchCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_search WHERE article_id = ?`, article.ID).Scan(&searchCount); err != nil {
		t.Fatal(err)
	}
	if searchCount != 2 {
		t.Errorf("search rows = %d, want 2", searchCount)
	}
}

func TestPublishEditCreatesVersion2AndReplacesSearchRows(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	article, err := store.Publish(context.Background(), ArticleInput{
		CategoryID: cat.ID,
		Title:      "Changing the toner",
		BodyHTML:   "<p>Old instructions about the cartridge.</p>",
	}, personID)
	if err != nil {
		t.Fatalf("Publish (create): %v", err)
	}

	edited, err := store.Publish(context.Background(), ArticleInput{
		ID:              article.ID,
		CategoryID:      cat.ID,
		Title:           "Changing the toner",
		BodyHTML:        "<p>New instructions about the drum.</p>",
		ExpectedVersion: article.VersionNo,
	}, personID)
	if err != nil {
		t.Fatalf("Publish (edit): %v", err)
	}
	if edited.ID != article.ID {
		t.Errorf("edit changed the article id: got %d, want %d", edited.ID, article.ID)
	}
	if edited.VersionNo != 2 {
		t.Errorf("VersionNo = %d, want 2", edited.VersionNo)
	}

	var versionCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_article_versions WHERE article_id = ?`, article.ID).Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if versionCount != 2 {
		t.Errorf("version rows = %d, want 2", versionCount)
	}

	// The removed word no longer matches; the new one does (SPEC gate 1.30).
	var cartridgeCount, drumCount int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_search WHERE article_id = ? AND body LIKE '%cartridge%'`, article.ID).Scan(&cartridgeCount)
	sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_search WHERE article_id = ? AND body LIKE '%drum%'`, article.ID).Scan(&drumCount)
	if cartridgeCount != 0 {
		t.Error("old word 'cartridge' still indexed after edit")
	}
	if drumCount != 1 {
		t.Error("new word 'drum' not indexed after edit")
	}
}

// TestPublishRollsBackFullyOnError injects a failure partway through
// Publish's transaction (an edit whose category_id doesn't exist, which
// kb_article_versions.category_id's foreign key rejects, a step that runs
// after the kb_articles row has already been written) and checks that
// nothing from the attempt survives — not the article, not a version row,
// not a search row. kb_search is an FTS5 virtual table, and SQLite can't
// put a trigger on one, so a constraint on it can't be used to force the
// failure directly; a foreign-key violation later in the same transaction
// proves the same all-or-nothing guarantee.
func TestPublishRollsBackFullyOnError(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	article, err := store.Publish(context.Background(), ArticleInput{
		CategoryID: cat.ID,
		Title:      "Printer help",
		BodyHTML:   "<p>Original text.</p>",
	}, personID)
	if err != nil {
		t.Fatalf("Publish (create): %v", err)
	}

	const missingCategoryID = 999999
	_, err = store.Publish(context.Background(), ArticleInput{
		ID:              article.ID,
		CategoryID:      missingCategoryID,
		Title:           "Printer help",
		BodyHTML:        "<p>Edited text that must not be saved.</p>",
		ExpectedVersion: article.VersionNo,
	}, personID)
	if err == nil {
		t.Fatal("Publish with a missing category succeeded, want an error")
	}

	var title string
	var categoryID int64
	var versionNo int
	if err := sqlDB.QueryRow(`SELECT title, category_id, version_no FROM kb_articles WHERE id = ?`, article.ID).
		Scan(&title, &categoryID, &versionNo); err != nil {
		t.Fatalf("article vanished after rollback: %v", err)
	}
	if title != "Printer help" || categoryID != cat.ID || versionNo != 1 {
		t.Errorf("article after failed edit = (%q, %d, v%d), want the original (Printer help, %d, v1)", title, categoryID, versionNo, cat.ID)
	}

	var versionCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_article_versions WHERE article_id = ?`, article.ID).Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if versionCount != 1 {
		t.Errorf("version rows = %d, want 1 (only the original)", versionCount)
	}

	var searchBody string
	if err := sqlDB.QueryRow(`SELECT body FROM kb_search WHERE article_id = ? AND block_id = 1`, article.ID).Scan(&searchBody); err != nil {
		t.Fatal(err)
	}
	if searchBody != "Original text." {
		t.Errorf("search row body = %q, want the original text unchanged", searchBody)
	}
}

func TestPublishRejectsEmptyTitle(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)

	store := &ArticleStore{DB: sqlDB}
	_, err = store.Publish(context.Background(), ArticleInput{
		CategoryID: cat.ID,
		Title:      "   ",
		BodyHTML:   "<p>Body</p>",
	}, personID)
	if err != ErrArticleEmptyTitle {
		t.Errorf("Publish(blank title) error = %v, want ErrArticleEmptyTitle", err)
	}
}

// TestPublishDetectsVersionConflict covers SPEC gate 1.21: a second
// publisher whose edit started from a now-stale version_no is refused,
// without touching the article.
func TestPublishDetectsVersionConflict(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	article, err := store.Publish(context.Background(), ArticleInput{
		CategoryID: cat.ID, Title: "Toner", BodyHTML: "<p>original</p>",
	}, personID)
	if err != nil {
		t.Fatalf("Publish (create): %v", err)
	}

	// First editor publishes based on version 1: succeeds, becomes v2.
	if _, err := store.Publish(context.Background(), ArticleInput{
		ID: article.ID, CategoryID: cat.ID, Title: "Toner", BodyHTML: "<p>first edit</p>",
		ExpectedVersion: 1,
	}, personID); err != nil {
		t.Fatalf("Publish (first editor): %v", err)
	}

	// Second editor also started from version 1, which is now stale.
	_, err = store.Publish(context.Background(), ArticleInput{
		ID: article.ID, CategoryID: cat.ID, Title: "Toner", BodyHTML: "<p>second edit</p>",
		ExpectedVersion: 1,
	}, personID)
	if err != ErrVersionConflict {
		t.Errorf("Publish (stale version) error = %v, want ErrVersionConflict", err)
	}

	current, err := store.Get(article.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if current.BodyHTML != `<p data-b="1">first edit</p>` {
		t.Errorf("article after refused conflict = %q, want the first editor's text unchanged", current.BodyHTML)
	}
}

// TestPublishOverrideBypassesConflictAndKeepsBothInHistory covers SPEC
// gate 1.21's "Publish mine anyway": it saves despite the stale version,
// and both edits remain in History.
func TestPublishOverrideBypassesConflictAndKeepsBothInHistory(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	article, err := store.Publish(context.Background(), ArticleInput{
		CategoryID: cat.ID, Title: "Toner", BodyHTML: "<p>original</p>",
	}, personID)
	if err != nil {
		t.Fatalf("Publish (create): %v", err)
	}
	if _, err := store.Publish(context.Background(), ArticleInput{
		ID: article.ID, CategoryID: cat.ID, Title: "Toner", BodyHTML: "<p>first edit</p>",
		ExpectedVersion: 1,
	}, personID); err != nil {
		t.Fatalf("Publish (first editor): %v", err)
	}

	overridden, err := store.Publish(context.Background(), ArticleInput{
		ID: article.ID, CategoryID: cat.ID, Title: "Toner", BodyHTML: "<p>second edit</p>",
		ExpectedVersion: 1, Override: true,
	}, personID)
	if err != nil {
		t.Fatalf("Publish (override): %v", err)
	}
	if overridden.BodyHTML != `<p data-b="1">second edit</p>` {
		t.Errorf("article after override = %q, want the second editor's text", overridden.BodyHTML)
	}

	history, err := store.History(article.ID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("History rows = %d, want 3 (created, first edit, overriding edit)", len(history))
	}
}

func TestListByCategoryOnlyPublished(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	published, err := store.Publish(context.Background(), ArticleInput{
		CategoryID: cat.ID, Title: "Published one", BodyHTML: "<p>x</p>",
	}, personID)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO kb_articles (category_id, title, status, created_at, updated_at) VALUES (?, 'Archived one', 'archived', '', '')`,
		cat.ID,
	); err != nil {
		t.Fatalf("seed archived article: %v", err)
	}

	articles, err := store.ListByCategory(cat.ID)
	if err != nil {
		t.Fatalf("ListByCategory: %v", err)
	}
	if len(articles) != 1 || articles[0].ID != published.ID {
		t.Errorf("ListByCategory = %+v, want only the published article", articles)
	}
}
