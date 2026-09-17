//go:build integration

package kb

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// TestPublishArticleHTTPAppearsInCategory covers SPEC gate 1.14: choosing a
// category, typing a title and content, and publishing makes the article
// appear in its category straight away.
func TestPublishArticleHTTPAppearsInCategory(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	postForm(t, client, ts, "/kb/categories", url.Values{"name": {"Printers"}}).Body.Close()
	var categoryID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'Printers'`).Scan(&categoryID); err != nil {
		t.Fatalf("find category: %v", err)
	}

	resp := postForm(t, client, ts, "/kb/articles", url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)},
		"title":       {"Changing the toner"},
		"body_html":   {"<p>Open the front cover and pull the cartridge out.</p>"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /kb/articles status = %d, want 200 (following the redirect)", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "Changing the toner") {
		t.Errorf("article page missing its own title; got:\n%s", body)
	}

	categoryPage := readBody(t, mustGet(t, client, ts, "/kb/categories/"+strconv.FormatInt(categoryID, 10)))
	if !strings.Contains(categoryPage, "Changing the toner") {
		t.Errorf("category page missing the newly published article; got:\n%s", categoryPage)
	}
}

// TestEditArticleHTTPRecordsSecondVersion covers SPEC gate 1.22: every
// publish is recorded, including edits to an existing article.
func TestEditArticleHTTPRecordsSecondVersion(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	postForm(t, client, ts, "/kb/categories", url.Values{"name": {"Printers"}}).Body.Close()
	var categoryID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'Printers'`).Scan(&categoryID); err != nil {
		t.Fatalf("find category: %v", err)
	}

	postForm(t, client, ts, "/kb/articles", url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)},
		"title":       {"Changing the toner"},
		"body_html":   {"<p>Original text.</p>"},
	}).Body.Close()

	var articleID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_articles WHERE title = 'Changing the toner'`).Scan(&articleID); err != nil {
		t.Fatalf("find article: %v", err)
	}

	postForm(t, client, ts, "/kb/articles", url.Values{
		"id":          {strconv.FormatInt(articleID, 10)},
		"category_id": {strconv.FormatInt(categoryID, 10)},
		"title":       {"Changing the toner"},
		"body_html":   {"<p>Edited text.</p>"},
		"version_no":  {"1"},
	}).Body.Close()

	var versionCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_article_versions WHERE article_id = ?`, articleID).Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if versionCount != 2 {
		t.Errorf("version rows = %d, want 2", versionCount)
	}

	articlePage := readBody(t, mustGet(t, client, ts, "/kb/articles/"+strconv.FormatInt(articleID, 10)))
	if !strings.Contains(articlePage, "Edited text.") {
		t.Errorf("article page doesn't show the edited text; got:\n%s", articlePage)
	}
}

// TestPublishArticleHTTPConflictReturnsEditorNotRedirect covers SPEC gate
// 1.21: a stale version_no gets the editor back (with the posted text and
// a "Publish mine anyway" retry), not the usual redirect to the article.
func TestPublishArticleHTTPConflictReturnsEditorNotRedirect(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	postForm(t, client, ts, "/kb/categories", url.Values{"name": {"Printers"}}).Body.Close()
	var categoryID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'Printers'`).Scan(&categoryID); err != nil {
		t.Fatalf("find category: %v", err)
	}
	postForm(t, client, ts, "/kb/articles", url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)}, "title": {"Toner"}, "body_html": {"<p>original</p>"},
	}).Body.Close()
	var articleID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_articles WHERE title = 'Toner'`).Scan(&articleID); err != nil {
		t.Fatalf("find article: %v", err)
	}

	// One editor publishes based on version 1, becoming version 2.
	postForm(t, client, ts, "/kb/articles", url.Values{
		"id": {strconv.FormatInt(articleID, 10)}, "category_id": {strconv.FormatInt(categoryID, 10)},
		"title": {"Toner"}, "body_html": {"<p>first edit</p>"}, "version_no": {"1"},
	}).Body.Close()

	// A second editor, still holding version 1, tries to publish.
	resp := postForm(t, client, ts, "/kb/articles", url.Values{
		"id": {strconv.FormatInt(articleID, 10)}, "category_id": {strconv.FormatInt(categoryID, 10)},
		"title": {"Toner"}, "body_html": {"<p>second edit</p>"}, "version_no": {"1"},
	})
	if resp.Request.URL.Path != "/kb/articles" {
		t.Errorf("conflict response URL = %s, want to stay on /kb/articles (no redirect)", resp.Request.URL.Path)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "Someone else changed this article while you were editing.") {
		t.Errorf("response missing the exact conflict message; got:\n%s", body)
	}
	if !strings.Contains(body, "second edit") {
		t.Errorf("response lost the second editor's text; got:\n%s", body)
	}
	if !strings.Contains(body, "Publish mine anyway") {
		t.Errorf("response missing the override button; got:\n%s", body)
	}

	var currentBody string
	if err := sqlDB.QueryRow(`SELECT body_html FROM kb_articles WHERE id = ?`, articleID).Scan(&currentBody); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(currentBody, "first edit") {
		t.Errorf("article body after refused conflict = %q, want the first editor's text unchanged", currentBody)
	}
}

func mustGet(t *testing.T, client *http.Client, ts *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := client.Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}
