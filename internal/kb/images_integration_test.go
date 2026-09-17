//go:build integration

package kb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stas-comp/comphq/internal/kb/images"
)

func postBytes(t *testing.T, client *http.Client, ts *httptest.Server, path string, data []byte, origin string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "image/png")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

var testPNG = append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, bytes.Repeat([]byte{0}, 20)...)

func TestUploadImageHTTPAcceptsExactly20MB(t *testing.T) {
	ts, _, dataDir := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	data := append(append([]byte{}, testPNG...), make([]byte, images.MaxBytes-len(testPNG))...)
	resp := postBytes(t, client, ts, "/kb/images", data, ts.URL)
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 for exactly 20MB; body: %s", resp.StatusCode, body)
	}

	var decoded struct {
		Src string `json:"src"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, body)
	}
	if !strings.HasPrefix(decoded.Src, "/images/") {
		t.Errorf("src = %q, want a /images/ path", decoded.Src)
	}

	filename := strings.TrimPrefix(decoded.Src, "/images/")
	sha, _, _ := strings.Cut(filename, ".")
	onDisk := filepath.Join(dataDir, "images", sha[:2], filename)
	if _, err := os.Stat(onDisk); err != nil {
		t.Errorf("uploaded file not found on disk at %s: %v", onDisk, err)
	}
}

func TestUploadImageHTTPRejects20MBPlus1(t *testing.T) {
	ts, _, _ := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	data := append(append([]byte{}, testPNG...), make([]byte, images.MaxBytes-len(testPNG)+1)...)
	resp := postBytes(t, client, ts, "/kb/images", data, ts.URL)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for 20MB+1", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "20 MB") {
		t.Errorf("body = %q, want a plain message mentioning the 20 MB limit", body)
	}
}

func TestServeImageHTTPHeaders(t *testing.T) {
	ts, _, _ := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	resp := postBytes(t, client, ts, "/kb/images", testPNG, ts.URL)
	var decoded struct {
		Src string `json:"src"`
	}
	if err := json.Unmarshal([]byte(readBody(t, resp)), &decoded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	imgResp, err := client.Get(ts.URL + decoded.Src)
	if err != nil {
		t.Fatalf("GET %s: %v", decoded.Src, err)
	}
	defer imgResp.Body.Close()

	if imgResp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", imgResp.StatusCode)
	}
	if got := imgResp.Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", got)
	}
	if got := imgResp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := imgResp.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want immutable caching", got)
	}
}

func TestUploadImageHTTPRefusesWrongOrigin(t *testing.T) {
	ts, _, _ := newTestServerWithDataDir(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	resp := postBytes(t, client, ts, "/kb/images", testPNG, "https://evil.example")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403 for a mismatched Origin", resp.StatusCode)
	}
}
