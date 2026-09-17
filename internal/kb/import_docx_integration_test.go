//go:build integration

package kb

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// docxFixturesDir mirrors internal/kb/docx's own fixturesDir: this
// package's tests reuse the same committed, generator-produced .docx
// fixtures rather than hand-building or duplicating them.
func docxFixturesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return filepath.Join(root, "e2e", "fixtures", "docx")
}

func readDocxFixture(t *testing.T, relPath string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(docxFixturesDir(t), relPath))
	if err != nil {
		t.Fatalf("read fixture %s: %v", relPath, err)
	}
	return data
}

// postMultipartFile posts data as a multipart/form-data body with one
// file field named "file" — what editor.js's FormData upload produces.
func postMultipartFile(t *testing.T, client *http.Client, ts *httptest.Server, path, filename string, data []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", ts.URL)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// TestImportDocxHTTPConvertsStoresImagesNeverTouchesArticles covers SPEC
// B4: the endpoint stores pictures and returns {title, html, notes}, and
// never creates or changes an article.
func TestImportDocxHTTPConvertsStoresImagesNeverTouchesArticles(t *testing.T) {
	ts, sqlDB, dataDir := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	var articlesBefore int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_articles`).Scan(&articlesBefore); err != nil {
		t.Fatal(err)
	}

	resp := postMultipartFile(t, client, ts, "/kb/import/docx", "sample.docx", readDocxFixture(t, "sample.docx"))
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, body)
	}

	var decoded struct {
		Title string   `json:"title"`
		HTML  string   `json:"html"`
		Notes []string `json:"notes"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, body)
	}

	if decoded.Title != "Printer Supplies Handbook" {
		t.Errorf("title = %q, want %q", decoded.Title, "Printer Supplies Handbook")
	}
	if !strings.Contains(decoded.HTML, "<h2>Getting started</h2>") {
		t.Errorf("html missing a converted heading; got:\n%s", decoded.HTML)
	}
	if len(decoded.Notes) == 0 {
		t.Error("notes is empty, want at least the fixture's header/footer/comment/footnote/chart/shape/equation/EMF-picture entries")
	}

	// Pictures are stored through the same image-store function as an
	// upload, so their src is a real /images/... URL with a file on disk.
	imgIdx := strings.Index(decoded.HTML, `<img src="/images/`)
	if imgIdx < 0 {
		t.Fatalf("html has no /images/ img src; got:\n%s", decoded.HTML)
	}
	srcStart := imgIdx + len(`<img src="`)
	srcEnd := strings.Index(decoded.HTML[srcStart:], `"`)
	src := decoded.HTML[srcStart : srcStart+srcEnd]
	filename := strings.TrimPrefix(src, "/images/")
	sha, _, _ := strings.Cut(filename, ".")
	onDisk := filepath.Join(dataDir, "images", sha[:2], filename)
	if _, err := os.Stat(onDisk); err != nil {
		t.Errorf("imported image not found on disk at %s: %v", onDisk, err)
	}

	var articlesAfter int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM kb_articles`).Scan(&articlesAfter); err != nil {
		t.Fatal(err)
	}
	if articlesAfter != articlesBefore {
		t.Errorf("article count changed from %d to %d; import must never create or change an article", articlesBefore, articlesAfter)
	}
}

// TestImportDocxHTTPGoogleDocsFixtureImports covers SPEC gate 1.45: "A
// .docx downloaded from Google Docs imports too."
func TestImportDocxHTTPGoogleDocsFixtureImports(t *testing.T) {
	ts, _, _ := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	resp := postMultipartFile(t, client, ts, "/kb/import/docx", "googledocs.docx", readDocxFixture(t, "googledocs.docx"))
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, body)
	}
	var decoded struct{ Title string }
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, body)
	}
	if decoded.Title != "Office Wi-Fi Guide" {
		t.Errorf("title = %q, want %q", decoded.Title, "Office Wi-Fi Guide")
	}
}

// TestImportDocxHTTPBadFixturesShowExactMessage covers SPEC gate 1.51.
func TestImportDocxHTTPBadFixturesShowExactMessage(t *testing.T) {
	ts, _, _ := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	for _, name := range []string{"bad/oldword.doc", "bad/protected.docx", "bad/file.pdf", "bad/truncated.docx"} {
		t.Run(name, func(t *testing.T) {
			resp := postMultipartFile(t, client, ts, "/kb/import/docx", filepath.Base(name), readDocxFixture(t, name))
			body := readBody(t, resp)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body: %s", resp.StatusCode, body)
			}
			if strings.TrimSpace(body) != docxUnreadableMessage {
				t.Errorf("body = %q, want the exact 1.51 message %q", body, docxUnreadableMessage)
			}
		})
	}
}

// TestImportDocxHTTPZipBombRejected covers the zip-bomb size limit,
// distinct from the unreadable-file message.
func TestImportDocxHTTPZipBombRejected(t *testing.T) {
	ts, _, _ := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	resp := postMultipartFile(t, client, ts, "/kb/import/docx", "zipbomb.docx", readDocxFixture(t, "bad/zipbomb.docx"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestImportDocxHTTPOversizeFileRejected covers SPEC: "A file over 50 MB
// is refused with a plain message."
func TestImportDocxHTTPOversizeFileRejected(t *testing.T) {
	ts, _, _ := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	oversized := make([]byte, 50*1024*1024+1)
	resp := postMultipartFile(t, client, ts, "/kb/import/docx", "huge.docx", oversized)
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "50 MB") {
		t.Errorf("body = %q, want a plain message mentioning the 50 MB limit", body)
	}
}

// TestImportDocxHTTPRefusesWrongOrigin covers SPEC B4: "the origin check
// applies."
func TestImportDocxHTTPRefusesWrongOrigin(t *testing.T) {
	ts, _, _ := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "sample.docx")
	fw.Write(readDocxFixture(t, "sample.docx"))
	w.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/kb/import/docx", &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", "https://evil.example")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403 for a mismatched Origin", resp.StatusCode)
	}
}
