//go:build integration

package settings

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/db"
	"github.com/stas-comp/comphq/internal/kb"
)

func newTestServer(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	for _, section := range []string{"app", "people", "kb"} {
		migrations, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			t.Fatalf("LoadMigrations(%s): %v", section, err)
		}
		if err := db.RunMigrations(sqlDB, "test", migrations); err != nil {
			t.Fatalf("RunMigrations(%s): %v", section, err)
		}
	}

	srv, err := app.NewServer(sqlDB, "test", t.TempDir(), false)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	srv.Registry().Add(kb.Section(srv))
	srv.Registry().Add(Section(srv))

	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return ts, sqlDB
}

func signIn(t *testing.T, client *http.Client, ts *httptest.Server) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/who/add", strings.NewReader(url.Values{"name": {"Sam"}, "next": {"/"}}.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}
	resp.Body.Close()
}

func postForm(t *testing.T, client *http.Client, ts *httptest.Server, path string, form url.Values) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// TestExportHTTPZipContainsIndexArticlesAndDatabase covers SPEC gate
// 1.34: the export zip has index.html, one page per published article
// grouped by category, archived articles under _Archived, and a fresh
// comphq.db.
func TestExportHTTPZipContainsIndexArticlesAndDatabase(t *testing.T) {
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
		"body_html":   {"<p>Pull it out.</p>"},
	}).Body.Close()

	postForm(t, client, ts, "/kb/articles", url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)},
		"title":       {"Old policy"},
		"body_html":   {"<p>Retired.</p>"},
	}).Body.Close()
	var archiveID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_articles WHERE title = 'Old policy'`).Scan(&archiveID); err != nil {
		t.Fatalf("find article to archive: %v", err)
	}
	postForm(t, client, ts, "/kb/articles/"+strconv.FormatInt(archiveID, 10)+"/archive", url.Values{}).Body.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/settings/export", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /settings/export: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/zip" {
		t.Errorf("Content-Type = %q, want application/zip", got)
	}
	if disposition := resp.Header.Get("Content-Disposition"); !strings.HasPrefix(disposition, "attachment;") || !strings.Contains(disposition, "comphq-export-") {
		t.Errorf("Content-Disposition = %q, want an attachment named comphq-export-...", disposition)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}

	paths := map[string]bool{}
	for _, f := range zr.File {
		paths[f.Name] = true
	}

	for _, want := range []string{
		"index.html",
		"comphq.db",
		"articles/Printers/Changing the toner.html",
		"articles/_Archived/Old policy.html",
	} {
		if !paths[want] {
			t.Errorf("zip missing %q; got %v", want, keys(paths))
		}
	}

	indexFile, err := zr.Open("index.html")
	if err != nil {
		t.Fatal(err)
	}
	indexBytes, err := io.ReadAll(indexFile)
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexBytes)
	if !strings.Contains(index, "Changing the toner") {
		t.Error("index.html doesn't mention the published article")
	}
	if !strings.Contains(index, "Old policy") {
		t.Error("index.html doesn't mention the archived article")
	}

	dbFile, err := zr.Open("comphq.db")
	if err != nil {
		t.Fatal(err)
	}
	dbBytes, err := io.ReadAll(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(dbBytes, []byte("SQLite format 3\x00")) {
		t.Error("comphq.db in the zip doesn't look like a real SQLite file")
	}
}

// TestExportHTTPRefusesCrossOriginGET isn't needed: GET requests never go
// through the origin check (SPEC B4 only guards state-changing POSTs).
// Instead, confirm the export route needs a signed-in person like every
// other page.
func TestExportHTTPRedirectsToPickerWithoutPerson(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{
		Jar:           mustCookieJar(t),
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	resp, err := client.Get(ts.URL + "/settings/export")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d (redirect to the picker)", resp.StatusCode, http.StatusFound)
	}
}

func mustCookieJar(t *testing.T) *cookiejar.Jar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return jar
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
