//go:build integration

package questions

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/db"
)

func newServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	for _, section := range []string{"app", "people", "questions"} {
		ms, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.RunMigrations(sqlDB, "test", ms); err != nil {
			t.Fatal(err)
		}
	}
	srv, err := app.NewServer(sqlDB, "test", t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	srv.Registry().Add(Section(srv))
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	post(t, client, ts, "/who/add", url.Values{"name": {"Sam"}, "next": {"/"}})
	return ts, client
}

func post(t *testing.T, c *http.Client, ts *httptest.Server, path string, form url.Values) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func get(t *testing.T, c *http.Client, ts *httptest.Server, path string) (int, string) {
	t.Helper()
	resp, err := c.Get(ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestPagesRenderAndFlowsWork(t *testing.T) {
	ts, c := newServer(t)
	if code, body := get(t, c, ts, "/questions"); code != 200 || !strings.Contains(body, "Ask a question") {
		t.Fatalf("GET /questions = %d %.200s", code, body)
	}
	code, body := post(t, c, ts, "/questions", url.Values{"text": {"Where is the stamp book?"}, "details": {"Line one\nLine two"}})
	if code != 200 || !strings.Contains(body, "Where is the stamp book?") || !strings.Contains(body, "Asked by Sam") {
		t.Fatalf("after asking = %d %.300s", code, body)
	}
	if code, body := get(t, c, ts, "/questions/1"); code != 200 || !strings.Contains(body, "Line one\nLine two") || !strings.Contains(body, "Mark answered") {
		t.Fatalf("GET /questions/1 = %d %.300s", code, body)
	}
	if code, body := post(t, c, ts, "/questions/1/answers", url.Values{"text": {"In the top drawer"}}); code != 200 || !strings.Contains(body, "In the top drawer") {
		t.Fatalf("answering = %d %.300s", code, body)
	}
	if _, body := post(t, c, ts, "/questions/1/answered", nil); !strings.Contains(body, "Answered · marked by Sam") {
		t.Fatalf("marking = %.300s", body)
	}
	if _, body := post(t, c, ts, "/questions/1/unanswered", nil); !strings.Contains(body, "Mark answered") {
		t.Fatalf("unmarking = %.300s", body)
	}
	if code, _ := get(t, c, ts, "/questions/99"); code != 404 {
		t.Errorf("an unknown question = %d, want 404", code)
	}
	if code, body := get(t, c, ts, "/questions/version"); code != 200 || !strings.Contains(body, `"version"`) {
		t.Errorf("version = %d %s", code, body)
	}
	// Every write is a POST under the origin check.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/questions", strings.NewReader("text=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://evil.example")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Errorf("a POST from another origin was accepted: %d", resp.StatusCode)
	}
	if resp, err := c.Get(ts.URL + "/questions/1/answered"); err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			t.Errorf("GET /questions/1/answered answered 200: a write must be a POST")
		}
	}
}
