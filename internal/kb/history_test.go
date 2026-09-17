package kb

import (
	"context"
	"testing"

	"github.com/stas-comp/comphq/internal/people"
)

// publishForHistoryTest publishes without worrying about SPEC gate 1.21's
// conflict check: these tests are about history/restore/archive, not
// conflicts, so it always fetches the article's current version_no first
// (a no-op query when id is 0, a new article) rather than making every
// caller track version numbers by hand.
func publishForHistoryTest(t *testing.T, store *ArticleStore, categoryID, id, personID int64, title, body string) Article {
	t.Helper()
	var expectedVersion int
	if id != 0 {
		if err := store.DB.QueryRow(`SELECT version_no FROM kb_articles WHERE id = ?`, id).Scan(&expectedVersion); err != nil {
			t.Fatalf("look up current version: %v", err)
		}
	}
	article, err := store.Publish(context.Background(), ArticleInput{
		ID: id, CategoryID: categoryID, Title: title, BodyHTML: body, ExpectedVersion: expectedVersion,
	}, personID)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return article
}

func TestHistoryListsVersionsNewestFirst(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	article := publishForHistoryTest(t, store, cat.ID, 0, personID, "Toner", "<p>v1</p>")
	publishForHistoryTest(t, store, cat.ID, article.ID, personID, "Toner", "<p>v2</p>")
	publishForHistoryTest(t, store, cat.ID, article.ID, personID, "Toner", "<p>v3</p>")

	history, err := store.History(article.ID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("History = %d rows, want 3", len(history))
	}
	if history[0].VersionNo != 3 || history[1].VersionNo != 2 || history[2].VersionNo != 1 {
		t.Errorf("History version order = [%d %d %d], want [3 2 1]", history[0].VersionNo, history[1].VersionNo, history[2].VersionNo)
	}
	if history[0].Action != "edited" || history[2].Action != "created" {
		t.Errorf("History actions = [%q ... %q], want created first, edited later", history[2].Action, history[0].Action)
	}
	for _, v := range history {
		if v.EditedByName != "Sam" {
			t.Errorf("EditedByName = %q, want Sam", v.EditedByName)
		}
	}
}

func TestVersionShowsContentAsItWas(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	article := publishForHistoryTest(t, store, cat.ID, 0, personID, "Toner", "<p>original</p>")
	publishForHistoryTest(t, store, cat.ID, article.ID, personID, "Toner", "<p>edited</p>")

	v1, err := store.Version(article.ID, 1)
	if err != nil {
		t.Fatalf("Version(1): %v", err)
	}
	if v1.BodyHTML != `<p data-b="1">original</p>` {
		t.Errorf("Version(1).BodyHTML = %q, want the original text", v1.BodyHTML)
	}

	current, err := store.Get(article.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if current.BodyHTML != `<p data-b="1">edited</p>` {
		t.Errorf("current BodyHTML = %q, want the edited text (Version shouldn't change it)", current.BodyHTML)
	}
}

func TestRestoreMakesOldVersionCurrentAndRecordsRestorer(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	author := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	article := publishForHistoryTest(t, store, cat.ID, 0, author, "Toner", "<p>original</p>")
	publishForHistoryTest(t, store, cat.ID, article.ID, author, "Toner", "<p>edited</p>")

	restorer, err := (&people.Store{DB: sqlDB}).Create("Robin")
	if err != nil {
		t.Fatalf("create restorer: %v", err)
	}

	restored, err := store.Restore(context.Background(), article.ID, 1, restorer.ID)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored.VersionNo != 3 {
		t.Errorf("VersionNo after restore = %d, want 3 (a new version, not a rewind)", restored.VersionNo)
	}
	if restored.BodyHTML != `<p data-b="1">original</p>` {
		t.Errorf("BodyHTML after restore = %q, want the original text back", restored.BodyHTML)
	}

	current, err := store.Get(article.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if current.BodyHTML != `<p data-b="1">original</p>` {
		t.Errorf("current article BodyHTML = %q, want the restored text", current.BodyHTML)
	}

	history, err := store.History(article.ID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if history[0].Action != "restored" || history[0].EditedByName != "Robin" {
		t.Errorf("newest history row = (%q, %q), want (restored, Robin)", history[0].Action, history[0].EditedByName)
	}
	if !history[0].RestoredFrom.Valid || history[0].RestoredFrom.Int64 != 1 {
		t.Errorf("RestoredFrom = %v, want valid 1", history[0].RestoredFrom)
	}

	var searchBody string
	if err := sqlDB.QueryRow(`SELECT body FROM kb_search WHERE article_id = ? AND block_id = 1`, article.ID).Scan(&searchBody); err != nil {
		t.Fatal(err)
	}
	if searchBody != "original" {
		t.Errorf("search body = %q, want the restored text indexed", searchBody)
	}
}

func TestArchiveRemovesFromCategoryAndSearchListsInArchived(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	article := publishForHistoryTest(t, store, cat.ID, 0, personID, "Toner", "<p>text</p>")

	if err := store.Archive(context.Background(), article.ID, personID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	catArticles, err := store.ListByCategory(cat.ID)
	if err != nil {
		t.Fatalf("ListByCategory: %v", err)
	}
	if len(catArticles) != 0 {
		t.Errorf("ListByCategory after archive = %+v, want empty", catArticles)
	}

	var searchCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_search WHERE article_id = ?`, article.ID).Scan(&searchCount); err != nil {
		t.Fatal(err)
	}
	if searchCount != 0 {
		t.Errorf("search rows after archive = %d, want 0", searchCount)
	}

	archived, err := store.ListArchived()
	if err != nil {
		t.Fatalf("ListArchived: %v", err)
	}
	if len(archived) != 1 || archived[0].ID != article.ID {
		t.Errorf("ListArchived = %+v, want the archived article", archived)
	}

	history, err := store.History(article.ID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if history[0].Action != "archived" {
		t.Errorf("newest history action = %q, want archived", history[0].Action)
	}
}

func TestUnarchiveRestoresCategoryAndSearch(t *testing.T) {
	sqlDB := openTestDB(t)
	catStore := &CategoryStore{DB: sqlDB}
	cat, err := catStore.Create("Printers")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}

	article := publishForHistoryTest(t, store, cat.ID, 0, personID, "Toner", "<p>text</p>")
	if err := store.Archive(context.Background(), article.ID, personID); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if err := store.Unarchive(context.Background(), article.ID, personID); err != nil {
		t.Fatalf("Unarchive: %v", err)
	}

	catArticles, err := store.ListByCategory(cat.ID)
	if err != nil {
		t.Fatalf("ListByCategory: %v", err)
	}
	if len(catArticles) != 1 {
		t.Errorf("ListByCategory after unarchive = %+v, want the article back", catArticles)
	}

	var searchCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_search WHERE article_id = ?`, article.ID).Scan(&searchCount); err != nil {
		t.Fatal(err)
	}
	if searchCount == 0 {
		t.Error("search rows after unarchive = 0, want them re-added")
	}

	archived, err := store.ListArchived()
	if err != nil {
		t.Fatalf("ListArchived: %v", err)
	}
	if len(archived) != 0 {
		t.Errorf("ListArchived after unarchive = %+v, want empty", archived)
	}
}
