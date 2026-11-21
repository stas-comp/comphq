//go:build integration

package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stas-comp/comphq/internal/brand"
)

// D-103: the same app, started with COMPHQ_BRAND=mc, is MC HQ everywhere a
// person sees the name, wears the crimson-and-gold colours (theme.css reads
// data-brand) and hands out MC HQ's own icon and manifest. Started without
// it, it is Comp HQ exactly as before.

func useBrand(t *testing.T, key string) {
	t.Helper()
	if err := brand.Set(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { brand.Set("") })
}

func TestMCBrandShowsMCHQEverywhere(t *testing.T) {
	useBrand(t, "mc")
	ts, _ := newTestServer(t)

	// The name picker is the first page anyone sees, and stands outside the frame.
	_, body := get(t, noRedirect(), ts.URL, "/who")
	page := string(body)
	for _, want := range []string{
		`<html lang="en" data-brand="mc">`,
		`<title>MC HQ — Who's using this computer?</title>`,
		`<div class="wordmark"><small>MC</small>`,
		`href="/static/theme/icons/mc/mchq.ico"`,
		`href="/static/theme/icons/mc/icon-192.png"`,
		`href="/static/app/manifest-mc.webmanifest"`,
		`<meta name="theme-color" content="#6b0f24">`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("/who lacks %s", want)
		}
	}
	for _, unwanted := range []string{"Comp HQ", "<small>Comp</small>", "comphq.ico", `content="#141b2d"`} {
		if strings.Contains(page, unwanted) {
			t.Errorf("/who still has %s", unwanted)
		}
	}

	resp, ico := get(t, noRedirect(), ts.URL, "/favicon.ico")
	if resp.StatusCode != 200 || !bytes.HasPrefix(ico, []byte{0, 0, 1, 0}) {
		t.Fatalf("/favicon.ico = %d, not an .ico", resp.StatusCode)
	}
	_, linked := get(t, noRedirect(), ts.URL, "/static/theme/icons/mc/mchq.ico")
	if !bytes.Equal(ico, linked) {
		t.Error("/favicon.ico isn't MC HQ's mchq.ico")
	}
	_, comp := get(t, noRedirect(), ts.URL, "/static/theme/icons/comphq.ico")
	if bytes.Equal(ico, comp) {
		t.Error("/favicon.ico is still Comp HQ's icon")
	}

	_, mbody := get(t, noRedirect(), ts.URL, "/static/app/manifest-mc.webmanifest")
	var m struct {
		Name        string `json:"name"`
		ShortName   string `json:"short_name"`
		Description string `json:"description"`
		ThemeColor  string `json:"theme_color"`
	}
	if err := json.Unmarshal(mbody, &m); err != nil {
		t.Fatal(err)
	}
	if m.Name != "MC HQ" || m.ShortName != "MC HQ" || !strings.Contains(m.Description, "MC Department") || m.ThemeColor != "#6b0f24" {
		t.Errorf("MC HQ's manifest is %+v", m)
	}

	// theme.css carries MC HQ's colours, keyed on the attribute above.
	_, css := get(t, noRedirect(), ts.URL, "/static/theme/theme.css")
	if !strings.Contains(string(css), `:root[data-brand="mc"]`) {
		t.Error(`theme.css has no :root[data-brand="mc"] block`)
	}
}

func TestDefaultBrandIsCompHQ(t *testing.T) {
	useBrand(t, "")
	ts, _ := newTestServer(t)
	_, body := get(t, noRedirect(), ts.URL, "/who")
	page := string(body)
	for _, want := range []string{
		`<html lang="en" data-brand="comp">`,
		`<title>Comp HQ — Who's using this computer?</title>`,
		`<div class="wordmark"><small>Comp</small>`,
		`href="/static/theme/icons/comphq.ico"`,
		`href="/static/app/manifest.webmanifest"`,
		`<meta name="theme-color" content="#141b2d">`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("/who lacks %s", want)
		}
	}
	if strings.Contains(page, "MC HQ") {
		t.Error("Comp HQ's /who mentions MC HQ")
	}
}
