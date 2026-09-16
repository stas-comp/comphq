//go:build integration

package app

import (
	"database/sql"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
)

func newTestServer(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	for _, section := range []string{"app", "people"} {
		migrations, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			t.Fatalf("LoadMigrations(%s): %v", section, err)
		}
		if err := db.RunMigrations(sqlDB, "test", migrations); err != nil {
			t.Fatalf("RunMigrations(%s): %v", section, err)
		}
	}

	srv, err := NewServer(sqlDB, "test", false)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return ts, sqlDB
}

func TestMiddlewareRedirectsGETWithoutPerson(t *testing.T) {
	ts, _ := newTestServer(t)

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "/who?next=") {
		t.Errorf("Location = %q, want a /who?next=... redirect preserving the destination", loc)
	}
}

func TestMiddlewareShowsPickerForPOSTWithoutPerson(t *testing.T) {
	ts, _ := newTestServer(t)

	form := url.Values{"title": {"whatever"}}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/kb/articles", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (the picker rendered directly)", resp.StatusCode)
	}
}

func TestPersonCookieAttributes(t *testing.T) {
	ts, _ := newTestServer(t)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}

	form := url.Values{"name": {"Sam"}, "next": {"/"}}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/who/add", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /who/add: %v", err)
	}
	resp.Body.Close()

	u, _ := url.Parse(ts.URL)
	var found *http.Cookie
	for _, c := range jar.Cookies(u) {
		if c.Name == "comphq_person" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("comphq_person cookie was not set")
	}

	// http.Client's cookiejar doesn't expose HttpOnly/SameSite/MaxAge from
	// the Set-Cookie response directly, so re-issue the request (without
	// following the redirect, so the direct response's own header is what
	// gets inspected) and read the raw header instead.
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/who/add", strings.NewReader(url.Values{"name": {"Alex"}, "next": {"/"}}.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("Origin", ts.URL)
	resp2, err := noRedirect.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	raw := strings.Join(resp2.Header.Values("Set-Cookie"), "; ")
	for _, want := range []string{"comphq_person=", "HttpOnly", "SameSite=Lax", "Max-Age=34560000"} {
		if !strings.Contains(raw, want) {
			t.Errorf("Set-Cookie header %q missing %q", raw, want)
		}
	}
}

func TestSecurityHeadersOnHTMLResponses(t *testing.T) {
	ts, _ := newTestServer(t)

	for _, path := range []string{"/who", "/does-not-exist"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()

		csp := resp.Header.Get("Content-Security-Policy")
		if !strings.Contains(csp, "default-src 'self'") {
			t.Errorf("%s: Content-Security-Policy = %q, missing default-src 'self'", path, csp)
		}
		if got := resp.Header.Get("Referrer-Policy"); got != "same-origin" {
			t.Errorf("%s: Referrer-Policy = %q, want same-origin", path, got)
		}
	}
}

func TestOriginCheckRejectsCrossOriginPOST(t *testing.T) {
	ts, _ := newTestServer(t)

	form := url.Values{"name": {"Sam"}, "next": {"/"}}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/who/add", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
}
