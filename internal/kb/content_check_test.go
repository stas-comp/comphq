package kb

import (
	"context"
	"testing"
)

func TestListContentCheckListsPublishedAndArchivedWithIssuesOnly(t *testing.T) {
	sqlDB := openTestDB(t)
	cat, err := (&CategoryStore{DB: sqlDB}).Create("IT")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}
	ctx := context.Background()

	if _, err := store.Publish(ctx, ArticleInput{CategoryID: cat.ID, Title: "Clean article", BodyHTML: "<p>All good.</p>"}, personID); err != nil {
		t.Fatalf("Publish(Clean article): %v", err)
	}

	if _, err := store.Publish(ctx, ArticleInput{
		CategoryID: cat.ID, Title: "Flagged article",
		BodyHTML: `<p>See:</p><div data-missing-kind="chart"></div>`,
	}, personID); err != nil {
		t.Fatalf("Publish(Flagged article): %v", err)
	}

	archivedFlagged, err := store.Publish(ctx, ArticleInput{
		CategoryID: cat.ID, Title: "Archived but still flagged",
		BodyHTML: `<div data-missing-kind="shape"></div>`,
	}, personID)
	if err != nil {
		t.Fatalf("Publish(Archived but still flagged): %v", err)
	}
	if err := store.Archive(ctx, archivedFlagged.ID, personID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	got, err := store.ListContentCheck(ctx)
	if err != nil {
		t.Fatalf("ListContentCheck: %v", err)
	}

	var titles []string
	for _, a := range got {
		titles = append(titles, a.Title)
	}
	want := map[string]bool{"Flagged article": true, "Archived but still flagged": true}
	if len(titles) != len(want) {
		t.Fatalf("ListContentCheck returned %v, want exactly %v", titles, want)
	}
	for _, title := range titles {
		if !want[title] {
			t.Errorf("unexpected article in results: %q", title)
		}
	}
}

func TestListContentCheckEmptyLibraryReturnsNone(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &ArticleStore{DB: sqlDB}

	got, err := store.ListContentCheck(context.Background())
	if err != nil {
		t.Fatalf("ListContentCheck: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListContentCheck on an empty library = %v, want none", got)
	}
}

func TestListContentCheckArticleNoLongerFlaggedAfterRepublish(t *testing.T) {
	sqlDB := openTestDB(t)
	cat, err := (&CategoryStore{DB: sqlDB}).Create("IT")
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	personID := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}
	ctx := context.Background()

	article, err := store.Publish(ctx, ArticleInput{
		CategoryID: cat.ID, Title: "Fixable article",
		BodyHTML: `<div data-missing-kind="chart"></div>`,
	}, personID)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	got, err := store.ListContentCheck(ctx)
	if err != nil {
		t.Fatalf("ListContentCheck (before fix): %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListContentCheck (before fix) = %v, want exactly the flagged article", got)
	}

	if _, err := store.Publish(ctx, ArticleInput{
		ID: article.ID, CategoryID: cat.ID, Title: "Fixable article",
		BodyHTML: "<p>The chart was removed.</p>", ExpectedVersion: article.VersionNo,
	}, personID); err != nil {
		t.Fatalf("Publish (fix): %v", err)
	}

	got, err = store.ListContentCheck(ctx)
	if err != nil {
		t.Fatalf("ListContentCheck (after fix): %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListContentCheck (after fix) = %v, want none", got)
	}
}
