//go:build integration

package settings

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// The rewritten Setup page (SPEC B11.4, gate 5.25): the ways to try, in
// order, each headed by the browser it is for, and an icon to download.

func TestSetupPageLeadsWithInstallAsAnApp(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	resp, err := client.Get(ts.URL + "/settings/setup")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	page := string(body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	// The order: install from the menu, then the command, then the taskbar,
	// then the blank-icon rescue.
	last := -1
	for _, id := range []string{`id="way-install"`, `id="way-shortcut"`, `id="way-taskbar"`, `id="way-icon"`} {
		i := strings.Index(page, id)
		if i < 0 {
			t.Fatalf("the page has no %s", id)
		}
		if i < last {
			t.Errorf("%s comes before the section that should precede it", id)
		}
		last = i
	}

	// Each route names the browser it is for.
	install := page[strings.Index(page, `id="way-install"`):strings.Index(page, `id="way-shortcut"`)]
	for _, want := range []string{"In Microsoft Edge", "In Google Chrome", "Install this site as an app", "Install page as app"} {
		if !strings.Contains(install, want) {
			t.Errorf("way 1 doesn't say %q", want)
		}
	}
	shortcut := page[strings.Index(page, `id="way-shortcut"`):strings.Index(page, `id="way-taskbar"`)]
	for _, want := range []string{"Google Chrome", "Microsoft Edge", "Brave", `id="chrome-command"`, `id="edge-command"`, `id="brave-command"`, "New</strong> → <strong>Shortcut"} {
		if !strings.Contains(shortcut, want) {
			t.Errorf("way 2 doesn't say %q", want)
		}
	}
	// The commands are built from the address the page was reached at.
	if !strings.Contains(shortcut, "--app=http://"+strings.TrimPrefix(ts.URL, "http://")+"/") {
		t.Errorf("way 2's commands don't use this server's own address (%s)", ts.URL)
	}
	// Each command has a Copy button.
	if n := strings.Count(page, `class="mini copy-button"`); n != 3 {
		t.Errorf("%d Copy buttons, want 3", n)
	}
	if !strings.Contains(page, `Pin to taskbar`) || !strings.Contains(page, `href="/settings/setup/icon"`) || !strings.Contains(page, `Download icon`) {
		t.Error("the page is missing the taskbar route or the Download icon button")
	}
	for _, want := range []string{"Properties", "Change Icon", "Browse"} {
		if !strings.Contains(page[strings.Index(page, `id="way-icon"`):], want) {
			t.Errorf("way 4 doesn't mention %q", want)
		}
	}
}

func TestSetupIconDownloadIsARealIcoNamedCompHQ(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts)

	resp, err := client.Get(ts.URL + "/settings/setup/icon")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if disp := resp.Header.Get("Content-Disposition"); !strings.HasPrefix(disp, `attachment; filename="Comp HQ.ico"`) {
		t.Errorf("Content-Disposition = %q, want an attachment named Comp HQ.ico", disp)
	}
	if got := resp.Header.Get("Content-Type"); got != "image/x-icon" {
		t.Errorf("Content-Type = %q, want image/x-icon", got)
	}
	if !bytes.HasPrefix(body, []byte{0, 0, 1, 0}) {
		t.Error("the download isn't an .ico file")
	}

	// It is the same icon the pages link, not a second copy that could drift.
	same, err := client.Get(ts.URL + "/static/theme/icons/comphq.ico")
	if err != nil {
		t.Fatal(err)
	}
	defer same.Body.Close()
	linked, _ := io.ReadAll(same.Body)
	if !bytes.Equal(body, linked) {
		t.Error("the downloadable icon differs from the linked comphq.ico")
	}
}

func TestSetupIconNeedsAPersonLikeEveryOtherSettingsPage(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(ts.URL + "/settings/setup/icon")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("with nobody signed in, status = %d, want a redirect to the picker", resp.StatusCode)
	}
}
