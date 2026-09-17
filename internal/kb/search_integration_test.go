//go:build integration

package kb

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// TestSearchJSONHTTPShapeAndEscaping covers the search.json endpoint the
// live panel (SPEC gate 1.26) polls: its JSON shape, and that a title or
// category containing HTML-significant characters is carried safely —
// encoding/json's default HTML-escaping protects the raw JSON payload,
// and search.js's own escapeHTML() protects the DOM once decoded, so the
// title and category here should round-trip to their real, unescaped
// text once a JSON decoder (standing in for the browser's fetch().json())
// reads them back.
func TestSearchJSONHTTPShapeAndEscaping(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	postForm(t, client, ts, "/kb/categories", url.Values{"name": {"Printers & Scanners"}}).Body.Close()
	var categoryID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'Printers & Scanners'`).Scan(&categoryID); err != nil {
		t.Fatalf("find category: %v", err)
	}

	postForm(t, client, ts, "/kb/articles", url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)},
		"title":       {"<script>Cartridges</script> supplies"},
		"body_html":   {"<p>Ask Sam about the toner cartridges.</p>"},
	}).Body.Close()
	var articleID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_articles WHERE body_html LIKE '%cartridges%'`).Scan(&articleID); err != nil {
		t.Fatalf("find article: %v", err)
	}

	resp := mustGet(t, client, ts, "/kb/search.json?q=toner")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	body := readBody(t, resp)

	// The raw wire format HTML-escapes '<', '>' and '&' (Go's
	// json.Encoder default), so the payload is inert even before a
	// client-side JSON parser touches it. Comparing against json.Marshal's
	// own output for a lone '<' or '&', rather than a hand-typed escape
	// sequence, is what that character actually turns into on the wire.
	escapedLT, _ := json.Marshal("<")
	escapedAmp, _ := json.Marshal("&")
	if strings.Contains(body, "<script>") {
		t.Errorf("response body contains an unescaped <script>; got:\n%s", body)
	}
	if !strings.Contains(body, strings.Trim(string(escapedLT), `"`)) {
		t.Errorf("response body wasn't HTML-escaped by json.Marshal; got:\n%s", body)
	}
	if !strings.Contains(body, strings.Trim(string(escapedAmp), `"`)) {
		t.Errorf("response body's '&' wasn't HTML-escaped by json.Marshal; got:\n%s", body)
	}

	var parsed struct {
		Query   string `json:"query"`
		Results []struct {
			ArticleID int64  `json:"articleId"`
			Title     string `json:"title"`
			Category  string `json:"category"`
			Snippet   string `json:"snippet"`
			URL       string `json:"url"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %v\nbody:\n%s", err, body)
	}
	if parsed.Query != "toner" {
		t.Errorf("query = %q, want %q", parsed.Query, "toner")
	}
	if len(parsed.Results) != 1 {
		t.Fatalf("results = %+v, want 1", parsed.Results)
	}

	r := parsed.Results[0]
	if r.ArticleID != articleID {
		t.Errorf("articleId = %d, want %d", r.ArticleID, articleID)
	}
	if r.Title != "<script>Cartridges</script> supplies" {
		t.Errorf("title = %q, want the article's real title once JSON-decoded", r.Title)
	}
	if r.Category != "Printers & Scanners" {
		t.Errorf("category = %q, want %q", r.Category, "Printers & Scanners")
	}
	if !strings.Contains(r.Snippet, "<mark>toner</mark>") {
		t.Errorf("snippet = %q, want it to contain <mark>toner</mark>", r.Snippet)
	}
	wantURLPrefix := "/kb/articles/" + strconv.FormatInt(articleID, 10) + "?q=toner#b-"
	if !strings.HasPrefix(r.URL, wantURLPrefix) {
		t.Errorf("url = %q, want prefix %q", r.URL, wantURLPrefix)
	}
}

// TestSearchJSONHTTPExcludesArchivedArticle covers SPEC gate 1.31:
// archived articles never appear in search results.
func TestSearchJSONHTTPExcludesArchivedArticle(t *testing.T) {
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
		"body_html":   {"<p>Ask Sam about the toner cartridges.</p>"},
	}).Body.Close()
	var articleID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_articles WHERE title = 'Toner'`).Scan(&articleID); err != nil {
		t.Fatalf("find article: %v", err)
	}

	postForm(t, client, ts, "/kb/articles/"+strconv.FormatInt(articleID, 10)+"/archive", url.Values{}).Body.Close()

	resp := mustGet(t, client, ts, "/kb/search.json?q=toner")
	var parsed struct {
		Results []struct {
			ArticleID int64 `json:"articleId"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(readBody(t, resp)), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Results) != 0 {
		t.Errorf("results = %+v, want none (article is archived)", parsed.Results)
	}
}
