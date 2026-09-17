//go:build integration

package kb

import (
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
)

func mustCookieJar(t *testing.T) *cookiejar.Jar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return jar
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func newTestServer(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	ts, sqlDB, _ := newTestServerWithDataDir(t)
	return ts, sqlDB
}

func newTestServerWithDataDir(t *testing.T) (*httptest.Server, *sql.DB, string) {
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

	dataDir := t.TempDir()
	srv, err := app.NewServer(sqlDB, "test", dataDir, false)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	srv.Registry().Add(Section(srv))

	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return ts, sqlDB, dataDir
}

// postForm issues an authenticated (origin-matching), person-identified
// POST — every state-changing route needs both (SPEC B4).
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

func signIn(t *testing.T, client *http.Client, ts *httptest.Server) {
	t.Helper()
	resp := postForm(t, client, ts, "/who/add", url.Values{"name": {"Sam"}, "next": {"/"}})
	resp.Body.Close()
}

func TestDeleteCategoryRefusedWithArticlesHTTP(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	resp := postForm(t, client, ts, "/kb/categories", url.Values{"name": {"Printers"}})
	resp.Body.Close()

	var categoryID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'Printers'`).Scan(&categoryID); err != nil {
		t.Fatalf("find category: %v", err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO kb_articles (category_id, title, status, created_at, updated_at) VALUES (?, 'A', 'published', '', '')`,
		categoryID,
	); err != nil {
		t.Fatalf("seed article: %v", err)
	}

	resp = postForm(t, client, ts, "/kb/categories/"+strconv.FormatInt(categoryID, 10)+"/delete", url.Values{})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (refused with a message, not an error page)", resp.StatusCode)
	}
	if !strings.Contains(body, "This category still has 1 article.") {
		t.Errorf("response body missing the refusal message; got:\n%s", body)
	}

	var stillExists int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_categories WHERE id = ?`, categoryID).Scan(&stillExists); err != nil {
		t.Fatal(err)
	}
	if stillExists != 1 {
		t.Error("category was deleted despite having an article")
	}
}

func TestReorderCategoriesRenumbersHTTP(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	for _, name := range []string{"A", "B", "C"} {
		postForm(t, client, ts, "/kb/categories", url.Values{"name": {name}}).Body.Close()
	}

	var bID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'B'`).Scan(&bID); err != nil {
		t.Fatal(err)
	}
	postForm(t, client, ts, "/kb/categories/"+strconv.FormatInt(bID, 10)+"/move-up", url.Values{}).Body.Close()

	rows, err := sqlDB.Query(`SELECT name FROM kb_categories ORDER BY sort_order`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var order []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		order = append(order, name)
	}
	if len(order) != 3 || order[0] != "B" || order[1] != "A" || order[2] != "C" {
		t.Errorf("order after moving B up = %v, want [B A C]", order)
	}
}
