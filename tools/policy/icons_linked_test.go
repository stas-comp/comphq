package policy

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Gates 5.22 and 5.23: what the pages link and what the manifest lists must be
// real files of the size they claim, so an icon can't be named and then not
// exist (which is how every shortcut came out blank before v1.2).
func TestLinkedAndManifestIconsExistAtTheirDeclaredSizes(t *testing.T) {
	root := repoRoot(t)
	staticFile := func(src string) string {
		if !strings.HasPrefix(src, "/static/") {
			t.Fatalf("%s isn't served from /static/", src)
		}
		return filepath.Join(root, "web", "static", filepath.FromSlash(strings.TrimPrefix(src, "/static/")))
	}
	checkPNG := func(where, src, sizes string) {
		data, err := os.ReadFile(staticFile(src))
		if err != nil {
			t.Errorf("%s names %s, which doesn't exist", where, src)
			return
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || format != "png" {
			t.Errorf("%s: %s isn't a PNG (%v)", where, src, err)
			return
		}
		if want := strconv.Itoa(cfg.Width) + "x" + strconv.Itoa(cfg.Height); sizes != want {
			t.Errorf("%s: %s is declared %s but is %s", where, src, sizes, want)
		}
	}

	manifestData, err := os.ReadFile(filepath.Join(root, "web", "static", "app", "manifest.webmanifest"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Icons []struct{ Src, Sizes, Type string } `json:"icons"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("manifest.webmanifest isn't JSON: %v", err)
	}
	if len(manifest.Icons) < len(pngSizes)+1 {
		t.Errorf("the manifest lists %d icons, want at least %d (every size, and the maskable)", len(manifest.Icons), len(pngSizes)+1)
	}
	for _, ic := range manifest.Icons {
		checkPNG("the manifest", ic.Src, ic.Sizes)
	}

	partsData, err := os.ReadFile(filepath.Join(root, "web", "templates", "app", "parts.html"))
	if err != nil {
		t.Fatal(err)
	}
	parts := string(partsData)
	start := strings.Index(parts, `{{define "app-head-icons"}}`)
	if start < 0 {
		t.Fatal(`parts.html has no "app-head-icons" partial`)
	}
	block := parts[start:]
	block = block[:strings.Index(block, "{{end}}")]
	linkRe := regexp.MustCompile(`<link rel="(icon|apple-touch-icon)"([^>]*)href="([^"]+)"`)
	found := 0
	for _, m := range linkRe.FindAllStringSubmatch(block, -1) {
		found++
		src := m[3]
		if _, err := os.Stat(staticFile(src)); err != nil {
			t.Errorf("the head links %s, which doesn't exist", src)
			continue
		}
		if strings.HasSuffix(src, ".png") {
			sizes := regexp.MustCompile(`sizes="(\d+x\d+)"`).FindStringSubmatch(m[2])
			declared := ""
			if sizes != nil {
				declared = sizes[1]
			} else if strings.Contains(m[1], "apple-touch-icon") {
				declared = "192x192" // the apple touch icon carries no sizes; it points at the 192
			}
			checkPNG("the head", src, declared)
		}
	}
	if found < 5 {
		t.Errorf("the head partial links %d icons, want the .ico, the PNG sizes and the apple touch icon", found)
	}
	if !strings.Contains(block, `rel="manifest"`) || !strings.Contains(block, `name="theme-color"`) {
		t.Error("the head partial links no manifest or theme colour")
	}

	// Every page that has a <head> of its own includes the partial (the test
	// harness page is exempt: it is never shown to a person).
	for _, rel := range []string{"web/templates/app/layout.html", "web/templates/people/who.html"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `{{template "app-head-icons"}}`) {
			t.Errorf("%s doesn't include the icon links; a page without them has no icon", rel)
		}
	}
}
