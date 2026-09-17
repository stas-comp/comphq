//go:build integration

package kb

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// TestHistoryHTTPListsVersionsNewestFirst covers SPEC gate 1.22.
func TestHistoryHTTPListsVersionsNewestFirst(t *testing.T) {
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
		"title":       {"Toner"},
		"body_html":   {"<p>v1</p>"},
	}).Body.Close()
	var articleID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_articles WHERE title = 'Toner'`).Scan(&articleID); err != nil {
		t.Fatalf("find article: %v", err)
	}
	postForm(t, client, ts, "/kb/articles", url.Values{
		"id": {strconv.FormatInt(articleID, 10)}, "category_id": {strconv.FormatInt(categoryID, 10)},
		"title": {"Toner"}, "body_html": {"<p>v2</p>"}, "version_no": {"1"},
	}).Body.Close()

	body := readBody(t, mustGet(t, client, ts, "/kb/articles/"+strconv.FormatInt(articleID, 10)+"/history"))
	v2Index := strings.Index(body, "Version 2")
	v1Index := strings.Index(body, "Version 1")
	if v2Index < 0 || v1Index < 0 {
		t.Fatalf("history page missing a version entry; got:\n%s", body)
	}
	if v2Index > v1Index {
		t.Errorf("Version 2 should appear before Version 1 (newest first); got:\n%s", body)
	}
	if !strings.Contains(body, "Sam") {
		t.Errorf("history page missing the editor's name; got:\n%s", body)
	}
}

// TestVersionAndRestoreHTTP covers SPEC gate 1.23: an old version displays
// as it was, and restoring it makes it current with a new history entry
// naming the restorer.
func TestVersionAndRestoreHTTP(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts) // signs in as "Sam"

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
	postForm(t, client, ts, "/kb/articles", url.Values{
		"id": {strconv.FormatInt(articleID, 10)}, "category_id": {strconv.FormatInt(categoryID, 10)},
		"title": {"Toner"}, "body_html": {"<p>edited</p>"}, "version_no": {"1"},
	}).Body.Close()

	versionPage := readBody(t, mustGet(t, client, ts, "/kb/articles/"+strconv.FormatInt(articleID, 10)+"/versions/1"))
	if !strings.Contains(versionPage, "original") {
		t.Errorf("version 1 page doesn't show the original text; got:\n%s", versionPage)
	}

	// A different person restores it.
	postForm(t, client, ts, "/who/add", url.Values{"name": {"Robin"}, "next": {"/"}}).Body.Close()
	resp := postForm(t, client, ts, "/kb/articles/"+strconv.FormatInt(articleID, 10)+"/versions/1/restore", url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restore status = %d, want 200 (following the redirect)", resp.StatusCode)
	}
	articlePage := readBody(t, resp)
	if !strings.Contains(articlePage, "original") {
		t.Errorf("article after restore doesn't show the original text; got:\n%s", articlePage)
	}

	historyPage := readBody(t, mustGet(t, client, ts, "/kb/articles/"+strconv.FormatInt(articleID, 10)+"/history"))
	if !strings.Contains(historyPage, "restored") || !strings.Contains(historyPage, "Robin") {
		t.Errorf("history missing a 'restored by Robin' entry; got:\n%s", historyPage)
	}
}

// TestArchiveAndUnarchiveHTTP covers SPEC gate 1.24.
func TestArchiveAndUnarchiveHTTP(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	postForm(t, client, ts, "/kb/categories", url.Values{"name": {"Printers"}}).Body.Close()
	var categoryID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'Printers'`).Scan(&categoryID); err != nil {
		t.Fatalf("find category: %v", err)
	}
	postForm(t, client, ts, "/kb/articles", url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)}, "title": {"Toner"}, "body_html": {"<p>text</p>"},
	}).Body.Close()
	var articleID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_articles WHERE title = 'Toner'`).Scan(&articleID); err != nil {
		t.Fatalf("find article: %v", err)
	}

	postForm(t, client, ts, "/kb/articles/"+strconv.FormatInt(articleID, 10)+"/archive", url.Values{}).Body.Close()

	categoryPage := readBody(t, mustGet(t, client, ts, "/kb/categories/"+strconv.FormatInt(categoryID, 10)))
	if strings.Contains(categoryPage, "Toner") {
		t.Errorf("archived article still listed in its category; got:\n%s", categoryPage)
	}
	archivedPage := readBody(t, mustGet(t, client, ts, "/kb/archived"))
	if !strings.Contains(archivedPage, "Toner") {
		t.Errorf("archived list missing the article; got:\n%s", archivedPage)
	}

	postForm(t, client, ts, "/kb/articles/"+strconv.FormatInt(articleID, 10)+"/unarchive", url.Values{}).Body.Close()

	categoryPage = readBody(t, mustGet(t, client, ts, "/kb/categories/"+strconv.FormatInt(categoryID, 10)))
	if !strings.Contains(categoryPage, "Toner") {
		t.Errorf("unarchived article missing from its category; got:\n%s", categoryPage)
	}
	archivedPage = readBody(t, mustGet(t, client, ts, "/kb/archived"))
	if strings.Contains(archivedPage, "Toner") {
		t.Errorf("unarchived article still listed as archived; got:\n%s", archivedPage)
	}
}

// TestNoDeleteRouteForArticles covers SPEC gate 1.25: nowhere in the app
// can an article be permanently deleted. Every article-related path
// registers only the methods this section actually needs (SPEC B4's
// enhanced ServeMux automatically answers 405 for a registered path given
// an unregistered method), so a DELETE to any of them must never succeed.
func TestNoDeleteRouteForArticles(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	postForm(t, client, ts, "/kb/categories", url.Values{"name": {"Printers"}}).Body.Close()
	var categoryID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'Printers'`).Scan(&categoryID); err != nil {
		t.Fatalf("find category: %v", err)
	}
	postForm(t, client, ts, "/kb/articles", url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)}, "title": {"Toner"}, "body_html": {"<p>text</p>"},
	}).Body.Close()
	var articleID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_articles WHERE title = 'Toner'`).Scan(&articleID); err != nil {
		t.Fatalf("find article: %v", err)
	}
	idStr := strconv.FormatInt(articleID, 10)

	paths := []string{
		"/kb/articles/" + idStr,
		"/kb/articles/" + idStr + "/edit",
		"/kb/articles/" + idStr + "/history",
		"/kb/articles/" + idStr + "/versions/1",
		"/kb/articles",
	}
	for _, path := range paths {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", ts.URL)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("DELETE %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			t.Errorf("DELETE %s = %d, want a non-success status (no delete route exists)", path, resp.StatusCode)
		}
	}

	var stillExists int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_articles WHERE id = ?`, articleID).Scan(&stillExists); err != nil {
		t.Fatal(err)
	}
	if stillExists != 1 {
		t.Error("article no longer exists after DELETE attempts")
	}
}
