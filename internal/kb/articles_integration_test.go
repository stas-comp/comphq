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

func mustGet(t *testing.T, client *http.Client, ts *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := client.Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}
