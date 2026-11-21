//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
	"github.com/stas-comp/comphq/internal/kb"
	"github.com/stas-comp/comphq/internal/people"
	"github.com/stas-comp/comphq/internal/tasks"
)

// Gates 8.11, 8.12, 8.18 with the real wiring: the Knowledge Base's own store
// behind Tasks' ArticleFinder, and the Tasks store behind the Knowledge Base's
// JobsUsing, joined only here (D-105, SPEC B2).
func TestJobsAndArticlesThroughTheRealWiring(t *testing.T) {
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	for _, section := range []string{"app", "people", "kb", "tasks"} {
		migrations, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.RunMigrations(sqlDB, "test", migrations); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	sam, err := (&people.Store{DB: sqlDB}).Create("Sam")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := (&kb.CategoryStore{DB: sqlDB}).Create("Mailing")
	if err != nil {
		t.Fatal(err)
	}
	articleStore := &kb.ArticleStore{DB: sqlDB}
	article, err := articleStore.Publish(ctx, kb.ArticleInput{CategoryID: cat.ID, Title: "Mailing SOP", BodyHTML: "<p>Steps.</p>"}, sam.ID)
	if err != nil {
		t.Fatal(err)
	}
	versionBefore := article.VersionNo
	var updatedBefore string
	sqlDB.QueryRow(`SELECT updated_at FROM kb_articles WHERE id = ?`, article.ID).Scan(&updatedBefore)

	store := &tasks.Store{DB: sqlDB, Articles: articleFinder{store: articleStore}}
	now := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	job, err := store.Create(ctx, tasks.CreateInput{Title: "Send the mailing"}, sam.ID, now)
	if err != nil {
		t.Fatal(err)
	}

	found, err := store.SearchArticles(ctx, "mail", job.ID)
	if err != nil || len(found) != 1 || found[0].Title != "Mailing SOP" || found[0].Category != "Mailing" {
		t.Fatalf("picker = %+v, %v", found, err)
	}
	if err := store.AddArticle(ctx, job.ID, article.ID, sam.ID, now); err != nil {
		t.Fatal(err)
	}
	if again, _ := store.SearchArticles(ctx, "mail", job.ID); len(again) != 0 {
		t.Errorf("a linked article is offered again: %+v", again)
	}

	// The article's side, through the other adapter.
	jobs := jobsUsingArticle{store: store}.JobsForArticle(ctx, article.ID)
	if len(jobs) != 1 || jobs[0].Title != "Send the mailing" || jobs[0].StageLabel != "To do" || jobs[0].Finished {
		t.Fatalf("jobs using the article = %+v", jobs)
	}

	// Gate 8.18: linking is not an edit of the article.
	after, err := articleStore.Get(article.ID)
	if err != nil {
		t.Fatal(err)
	}
	var updatedAfter string
	sqlDB.QueryRow(`SELECT updated_at FROM kb_articles WHERE id = ?`, article.ID).Scan(&updatedAfter)
	if after.VersionNo != versionBefore || updatedAfter != updatedBefore {
		t.Errorf("linking changed the article: version %d -> %d, updated %v -> %v", versionBefore, after.VersionNo, updatedBefore, updatedAfter)
	}
	var versions int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_article_versions WHERE article_id = ?`, article.ID).Scan(&versions)
	if versions != 1 {
		t.Errorf("%d version rows, want 1", versions)
	}

	// Archiving after linking: it stays on the job, marked.
	if err := articleStore.Archive(ctx, article.ID, sam.ID); err != nil {
		t.Fatal(err)
	}
	refs, err := store.ListReferences(ctx, job.ID)
	if err != nil || len(refs.Articles) != 1 || !refs.Articles[0].Archived || refs.Articles[0].Title != "Mailing SOP" {
		t.Fatalf("references = %+v, %v", refs, err)
	}
	if again, _ := store.SearchArticles(ctx, "mail", job.ID); len(again) != 0 {
		t.Errorf("an archived article is offered: %+v", again)
	}
}
