//go:build integration

package kb

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

func pngStub(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(tinyPNG)
	}))
}

// TestPublishArticleHTTPCopiesExternalImage covers SPEC gate 1.19: an
// external image reachable at publish time is copied onto the NAS and the
// article ends up pointing at the local copy, never the other host.
func TestPublishArticleHTTPCopiesExternalImage(t *testing.T) {
	stub := pngStub(t)
	defer stub.Close()
	t.Setenv("COMPHQ_TEST_MODE", "1")
	t.Setenv("COMPHQ_TEST_ALLOW_FETCH_HOST", stub.Listener.Addr().String())

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
		"title":       {"Diagram article"},
		"body_html":   {`<p>See below.</p><img src="` + stub.URL + `/x.png" alt="Diagram">`},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body:\n%s", resp.StatusCode, body)
	}
	if strings.Contains(body, stub.URL) {
		t.Errorf("published article still references the external host; body:\n%s", body)
	}
	if !strings.Contains(body, `src="/images/`) {
		t.Errorf("published article doesn't reference a local /images/ copy; body:\n%s", body)
	}

	var imageCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM images`).Scan(&imageCount); err != nil {
		t.Fatal(err)
	}
	if imageCount != 1 {
		t.Errorf("images rows = %d, want 1", imageCount)
	}
}

// TestPublishArticleHTTPStoresDataURLImage covers the data: half of SPEC
// gate 1.19.
func TestPublishArticleHTTPStoresDataURLImage(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	postForm(t, client, ts, "/kb/categories", url.Values{"name": {"Printers"}}).Body.Close()
	var categoryID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'Printers'`).Scan(&categoryID); err != nil {
		t.Fatalf("find category: %v", err)
	}

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(tinyPNG)
	resp := postForm(t, client, ts, "/kb/articles", url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)},
		"title":       {"Data URL article"},
		"body_html":   {`<img src="` + dataURL + `" alt="Inline">`},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body:\n%s", resp.StatusCode, body)
	}
	if !strings.Contains(body, `src="/images/`) {
		t.Errorf("published article doesn't reference a local /images/ copy; body:\n%s", body)
	}
}

// TestPublishArticleHTTPReportsUnreachableImage covers SPEC gate 1.19's
// failure path: the article still publishes, the failed image is marked
// with a placeholder holding the original URL, and the rest of the text
// is saved.
func TestPublishArticleHTTPReportsUnreachableImage(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	postForm(t, client, ts, "/kb/categories", url.Values{"name": {"Printers"}}).Body.Close()
	var categoryID int64
	if err := sqlDB.QueryRow(`SELECT id FROM kb_categories WHERE name = 'Printers'`).Scan(&categoryID); err != nil {
		t.Fatalf("find category: %v", err)
	}

	// No COMPHQ_TEST_ALLOW_FETCH_HOST override: this loopback address is
	// refused by the same SSRF guard that would refuse a real private
	// address, which is exactly the failure path this test wants.
	const unreachable = "http://127.0.0.1:1/unreachable.png"
	resp := postForm(t, client, ts, "/kb/articles", url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)},
		"title":       {"Unreachable image article"},
		"body_html":   {`<p>Kept text.</p><img src="` + unreachable + `" alt="Missing">`},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (publish still succeeds); body:\n%s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "Kept text.") {
		t.Errorf("published article lost its text; body:\n%s", body)
	}
	if !strings.Contains(body, `data-missing-src="`+unreachable+`"`) {
		t.Errorf("published article missing the placeholder for the failed image; body:\n%s", body)
	}
	if !strings.Contains(body, unreachable) {
		t.Errorf("response doesn't name the failed image %s; body:\n%s", unreachable, body)
	}
}
