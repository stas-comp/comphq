//go:build integration

package kb

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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

// importResult posts a fixture and returns the decoded response.
func importFixture(t *testing.T, client *http.Client, ts *httptest.Server, name string) (html string, notes []string) {
	t.Helper()
	resp := postMultipartFile(t, client, ts, "/kb/import/docx", name, readDocxFixture(t, name))
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: status = %d; body: %s", name, resp.StatusCode, body)
	}
	var decoded struct {
		HTML  string   `json:"html"`
		Notes []string `json:"notes"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("%s: decode: %v", name, err)
	}
	return decoded.HTML, decoded.Notes
}

// mediaHashes are the SHA-256 of the pictures stored in a fixture's
// word/media folder, by file name.
func mediaHashes(t *testing.T, data []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "word/media/") {
			continue
		}
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		sum := sha256.Sum256(b)
		out[strings.TrimPrefix(f.Name, "word/media/")] = hex.EncodeToString(sum[:])
	}
	return out
}

var storedImgRe = regexp.MustCompile(`<img src="/images/([0-9a-f]{64})\.[a-z]+"`)

// Gate 7.60 over HTTP: the twenty-five pictures come out as twenty-five real
// addresses, each the right picture, in order (at v1.3.0 the tenth on were
// corrupted into ".png0" and 404'd).
func TestImportDocxTwentyFivePicturesEachWithItsOwnAddress(t *testing.T) {
	ts, _, dataDir := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	html, notes := importFixture(t, client, ts, "pictures-25.docx")
	if len(notes) != 0 {
		t.Errorf("notes = %v, want none", notes)
	}
	matches := storedImgRe.FindAllStringSubmatch(html, -1)
	if len(matches) != 25 {
		t.Fatalf("%d stored pictures in the result, want 25; html:\n%s", len(matches), html)
	}
	hashes := mediaHashes(t, readDocxFixture(t, "pictures-25.docx"))
	for i, m := range matches {
		want := hashes["pic"+strconv.Itoa(i+1)+".png"]
		if m[1] != want {
			t.Errorf("picture %d has hash %s, want %s", i+1, m[1], want)
		}
		if _, err := os.Stat(filepath.Join(dataDir, "images", m[1][:2], m[1]+".png")); err != nil {
			t.Errorf("picture %d is not on disk: %v", i+1, err)
		}
	}
	if strings.Contains(html, "cid:") || strings.Contains(html, ".png0") {
		t.Errorf("a token or a corrupted address is left in the result: %s", html)
	}
}

// Gate 7.64: a picture that fails to save is marked in its place and counted,
// not silently dropped.
func TestImportDocxPictureThatFailsToSaveIsMarkedAndCounted(t *testing.T) {
	ts, _, dataDir := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)
	// A file where the images folder should be: every save fails.
	if err := os.WriteFile(filepath.Join(dataDir, "images"), []byte("not a folder"), 0o644); err != nil {
		t.Fatal(err)
	}

	html, notes := importFixture(t, client, ts, "picture-in-title.docx")
	if got := strings.Count(html, `data-missing-kind="picture"`); got != 2 {
		t.Errorf("%d placeholders, want 2 (one per picture that failed to save); html:\n%s", got, html)
	}
	if strings.Contains(html, "<img") {
		t.Errorf("a picture is claimed to be shown though nothing could be saved: %s", html)
	}
	if len(notes) != 2 || notes[0] != "a picture" {
		t.Errorf("notes = %v, want two entries 'a picture'", notes)
	}
}

// Gates 7.62, 7.67: every fixture imports with every <img> a real stored
// address (never a token), and the unusual-name pictures all arrive.
func TestImportDocxEveryFixtureLeavesNoTokenBehind(t *testing.T) {
	ts, _, dataDir := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	paths, _ := filepath.Glob(filepath.Join(docxFixturesDir(t), "*.docx"))
	samples, _ := filepath.Glob(filepath.Join(docxFixturesDir(t), "..", "..", "..", "samples", "word", "*.docx"))
	for _, path := range append(paths, samples...) {
		name := filepath.Base(path)
		data, _ := os.ReadFile(path)
		resp := postMultipartFile(t, client, ts, "/kb/import/docx", name, data)
		body := readBody(t, resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: status %d: %s", name, resp.StatusCode, body)
		}
		var decoded struct {
			HTML string `json:"html"`
		}
		if err := json.Unmarshal([]byte(body), &decoded); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(decoded.HTML, "cid:") {
			t.Errorf("%s: a token is left behind", name)
		}
		for _, m := range regexp.MustCompile(`<img [^>]*src="([^"]*)"`).FindAllStringSubmatch(decoded.HTML, -1) {
			if !strings.HasPrefix(m[1], "/images/") {
				t.Errorf("%s: an <img> with src %q", name, m[1])
				continue
			}
			file := strings.TrimPrefix(m[1], "/images/")
			if _, err := os.Stat(filepath.Join(dataDir, "images", file[:2], file)); err != nil {
				t.Errorf("%s: %s is not on disk: %v", name, m[1], err)
			}
		}
	}
	html, _ := importFixture(t, client, ts, "pictures-oddnames.docx")
	if got := len(storedImgRe.FindAllString(html, -1)); got != 3 {
		t.Errorf("pictures-oddnames: %d pictures arrived, want 3", got)
	}
	title, _ := importFixture(t, client, ts, "picture-in-title.docx")
	if got := len(storedImgRe.FindAllString(title, -1)); got != 2 {
		t.Errorf("picture-in-title: %d pictures arrived, want 2", got)
	}
}
