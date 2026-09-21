//go:build integration

package app

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Comp HQ's icon reaches the browser (SPEC B11, gates 5.22-5.23): every page
// links it, /favicon.ico answers with it, and the manifest an installed app
// reads names Comp HQ, opens at the Briefing and lists every size.

type headLink struct {
	rel, href, sizes, typ string
}

var linkTagRe = regexp.MustCompile(`<link\b[^>]*>`)

func attr(tag, name string) string {
	m := regexp.MustCompile(`\b` + name + `="([^"]*)"`).FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	return m[1]
}

func headLinks(page string) []headLink {
	head := page
	if i := strings.Index(page, "</head>"); i >= 0 {
		head = page[:i]
	}
	var out []headLink
	for _, tag := range linkTagRe.FindAllString(head, -1) {
		out = append(out, headLink{rel: attr(tag, "rel"), href: attr(tag, "href"), sizes: attr(tag, "sizes"), typ: attr(tag, "type")})
	}
	return out
}

func get(t *testing.T, client *http.Client, ts string, path string) (*http.Response, []byte) {
	t.Helper()
	resp, err := client.Get(ts + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, body
}

func noRedirect() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func TestFaviconAnswersWithTheIconAndAsksNobody(t *testing.T) {
	ts, _ := newTestServer(t)
	resp, body := get(t, noRedirect(), ts.URL, "/favicon.ico") // no cookie: nobody has said who they are
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /favicon.ico = %d (Location %q), want 200 with no sign-in", resp.StatusCode, resp.Header.Get("Location"))
	}
	if got := resp.Header.Get("Content-Type"); got != "image/x-icon" {
		t.Errorf("Content-Type = %q, want image/x-icon", got)
	}
	if !bytes.HasPrefix(body, []byte{0, 0, 1, 0}) {
		t.Error("the body isn't an .ico file")
	}
	if resp.Header.Get("Cache-Control") == "" {
		t.Error("the icon is sent with no caching header")
	}
	_, same := get(t, noRedirect(), ts.URL, "/static/theme/icons/comphq.ico")
	if !bytes.Equal(body, same) {
		t.Error("/favicon.ico and the linked comphq.ico are different files")
	}
}

func TestManifestNamesTheAppAndListsEverySizeAsAnImage(t *testing.T) {
	ts, _ := newTestServer(t)
	resp, body := get(t, noRedirect(), ts.URL, "/static/app/manifest.webmanifest")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("manifest = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/manifest+json") {
		t.Errorf("manifest Content-Type = %q, want application/manifest+json", ct)
	}

	var m struct {
		Name            string `json:"name"`
		ShortName       string `json:"short_name"`
		StartURL        string `json:"start_url"`
		Scope           string `json:"scope"`
		Display         string `json:"display"`
		ThemeColor      string `json:"theme_color"`
		BackgroundColor string `json:"background_color"`
		Icons           []struct {
			Src     string `json:"src"`
			Sizes   string `json:"sizes"`
			Type    string `json:"type"`
			Purpose string `json:"purpose"`
		} `json:"icons"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("the manifest isn't JSON: %v", err)
	}
	if m.Name != "Comp HQ" || m.ShortName != "Comp HQ" {
		t.Errorf("name/short_name = %q/%q, want Comp HQ", m.Name, m.ShortName)
	}
	// "/" is the Briefing (SPEC A8): an installed app opens where the app opens.
	if m.StartURL != "/" || m.Scope != "/" {
		t.Errorf("start_url/scope = %q/%q, want / and /", m.StartURL, m.Scope)
	}
	if m.Display != "standalone" {
		t.Errorf("display = %q, want standalone (its own window)", m.Display)
	}
	if m.ThemeColor != "#141b2d" { // --color-ink-navy
		t.Errorf("theme_color = %q, want the ink navy #141b2d", m.ThemeColor)
	}
	if m.BackgroundColor != "#f5f3ee" { // --color-paper
		t.Errorf("background_color = %q, want the paper #f5f3ee", m.BackgroundColor)
	}

	sizes := map[int]bool{}
	maskable := false
	for _, ic := range m.Icons {
		resp, data := get(t, noRedirect(), ts.URL, ic.Src)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("manifest icon %s = %d", ic.Src, resp.StatusCode)
			continue
		}
		if ct := resp.Header.Get("Content-Type"); ct != ic.Type {
			t.Errorf("manifest icon %s is served as %q, but the manifest says %q", ic.Src, ct, ic.Type)
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			t.Errorf("manifest icon %s isn't an image: %v", ic.Src, err)
			continue
		}
		if want := strconv.Itoa(cfg.Width) + "x" + strconv.Itoa(cfg.Height); ic.Sizes != want {
			t.Errorf("manifest icon %s claims %s but is %s", ic.Src, ic.Sizes, want)
		}
		sizes[cfg.Width] = true
		if ic.Purpose == "maskable" {
			maskable = true
		}
	}
	for _, want := range []int{16, 32, 48, 64, 128, 192, 256, 512} {
		if !sizes[want] {
			t.Errorf("the manifest lists no %dpx icon", want)
		}
	}
	if !maskable {
		t.Error("the manifest lists no maskable icon")
	}
}

// Gate 5.22: every page links the icon — the signed-in frame, the picker and
// the 404 alike. Each address it names answers with an image (or the manifest)
// of the size it says.
func TestEveryPageLinksIconsThatAnswer(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	signIn := func() {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/who/add", strings.NewReader(url.Values{"name": {"Sam"}, "next": {"/"}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", ts.URL)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	pages := []struct {
		path     string
		signedIn bool
	}{
		{"/who", false},             // the picker, before anybody has chosen
		{"/no-such-page-404", true}, // the framed 404
	}

	for _, page := range pages {
		path := page.path
		if page.signedIn {
			signIn()
		}
		resp, body := get(t, client, ts.URL, path)
		links := headLinks(string(body))
		var icons, apple, manifest, ico int
		for _, l := range links {
			switch {
			case l.rel == "icon" && strings.HasSuffix(l.href, ".ico"):
				ico++
			case l.rel == "icon":
				icons++
			case l.rel == "apple-touch-icon":
				apple++
			case l.rel == "manifest":
				manifest++
			default:
				continue
			}
			r, data := get(t, noRedirect(), ts.URL, l.href)
			if r.StatusCode != http.StatusOK {
				t.Errorf("%s (%d): <link rel=%s href=%s> = %d", path, resp.StatusCode, l.rel, l.href, r.StatusCode)
				continue
			}
			ct := r.Header.Get("Content-Type")
			switch l.rel {
			case "manifest":
				if !strings.HasPrefix(ct, "application/manifest+json") {
					t.Errorf("%s: manifest served as %q", path, ct)
				}
			default:
				if !strings.HasPrefix(ct, "image/") {
					t.Errorf("%s: %s served as %q, not an image", path, l.href, ct)
				}
				if strings.HasSuffix(l.href, ".png") {
					cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
					if err != nil {
						t.Errorf("%s: %s isn't a PNG: %v", path, l.href, err)
						continue
					}
					if want := strconv.Itoa(cfg.Width) + "x" + strconv.Itoa(cfg.Height); l.sizes != "" && l.sizes != want {
						t.Errorf("%s: %s is linked as %s but is %s", path, l.href, l.sizes, want)
					}
				}
			}
		}
		if ico != 1 || icons < 3 || apple != 1 || manifest != 1 {
			t.Errorf("%s links %d .ico, %d PNG icons, %d apple-touch-icon, %d manifest; want 1, at least 3, 1, 1 (%v)", path, ico, icons, apple, manifest, links)
		}
	}
}
