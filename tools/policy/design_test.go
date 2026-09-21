package policy

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The v1.1 design rules (SPEC B9.2, B9.9 layers 2 and 4, PLAN P4-02): the
// palette and type live only in theme.css, exactly as the spec's token
// table says, and nothing else in the app may invent a colour or a font.

// b92Colours is SPEC B9.2's colour table, value for value.
var b92Colours = map[string]string{
	"--color-ink-navy":           "#141b2d",
	"--color-ink-navy-light":     "#1f2842",
	"--color-ink-navy-3":         "#2c3656",
	"--color-signal-orange":      "#ff6b1a",
	"--color-signal-orange-deep": "#a94000",
	"--color-accent-soft":        "#ffe2cf",
	"--color-accent-wash":        "#fff3ea",
	"--color-paper":              "#f5f3ee",
	"--color-paper-muted":        "#ece9e1",
	"--color-card":               "#ffffff",
	"--color-ink-text":           "#1a1f2b",
	"--color-text-muted":         "#5a6072",
	"--color-border":             "#dedad0",
	"--color-border-strong":      "#cbc6b9",
	"--color-danger":             "#c62d2d",
	"--color-danger-soft":        "#fbe3e1",
}

// b92Fonts is the first family each font token names. The rest of the
// stack is fallbacks, which the spec doesn't fix.
var b92Fonts = map[string]string{
	"--font-display": `"Big Shoulders Display"`,
	"--font-body":    `"Atkinson Hyperlegible Next"`,
	"--font-mono":    `"IBM Plex Mono"`,
}

var (
	cssCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssBlockRe   = regexp.MustCompile(`([^{}]*)\{([^{}]*)\}`)
)

type cssDecl struct {
	selector string
	prop     string
	value    string
}

func readCSS(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return cssCommentRe.ReplaceAllString(string(data), "")
}

// cssDecls returns every declaration in every innermost block, so a rule
// inside @media is read the same as one outside it.
func cssDecls(css string) []cssDecl {
	var out []cssDecl
	for _, m := range cssBlockRe.FindAllStringSubmatch(css, -1) {
		selector := strings.TrimSpace(m[1])
		for _, d := range strings.Split(m[2], ";") {
			prop, value, ok := strings.Cut(d, ":")
			if !ok {
				continue
			}
			out = append(out, cssDecl{
				selector: selector,
				prop:     strings.ToLower(strings.TrimSpace(prop)),
				value:    strings.TrimSpace(value),
			})
		}
	}
	return out
}

func themeCSSPath(t *testing.T) string {
	return filepath.Join(repoRoot(t), "web", "static", "theme", "theme.css")
}

// sectionStylesheets is every stylesheet in the app except theme.css and
// third-party files: the ones that may only do layout.
func sectionStylesheets(t *testing.T) []string {
	t.Helper()
	root := filepath.Join(repoRoot(t), "web", "static")
	var found []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() || !strings.HasSuffix(path, ".css") || strings.HasPrefix(rel, "vendor/") || strings.HasPrefix(rel, "theme/") {
			return nil
		}
		found = append(found, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("found no section stylesheets under web/static")
	}
	return found
}

// SPEC B9.2 / B9.9 layer 2: theme.css carries every token, every value as
// the table writes it.
func TestThemeTokensMatchSpecTable(t *testing.T) {
	tokens := map[string]string{}
	for _, d := range cssDecls(readCSS(t, themeCSSPath(t))) {
		if d.selector == ":root" && strings.HasPrefix(d.prop, "--") {
			tokens[d.prop] = d.value
		}
	}
	for name, want := range b92Colours {
		got, ok := tokens[name]
		if !ok {
			t.Errorf("theme.css is missing %s", name)
			continue
		}
		if strings.ToLower(got) != want {
			t.Errorf("theme.css %s = %s, SPEC B9.2 says %s", name, got, want)
		}
	}
	for name, family := range b92Fonts {
		got, ok := tokens[name]
		if !ok {
			t.Errorf("theme.css is missing %s", name)
			continue
		}
		if !strings.HasPrefix(got, family) {
			t.Errorf("theme.css %s = %s, want a stack that starts with %s (SPEC B9.2)", name, got, family)
		}
	}
}

// Every @font-face in theme.css must point at a file that exists.
func TestThemeFontFilesExist(t *testing.T) {
	root := repoRoot(t)
	css := readCSS(t, themeCSSPath(t))
	urls := regexp.MustCompile(`url\("(/static/[^"]+\.woff2)"\)`).FindAllStringSubmatch(css, -1)
	if len(urls) != 3 {
		t.Errorf("theme.css loads %d font files, want 3 (Big Shoulders Display, Atkinson Hyperlegible Next, IBM Plex Mono)", len(urls))
	}
	for _, m := range urls {
		if _, err := os.Stat(filepath.Join(root, "web", filepath.FromSlash(m[1]))); err != nil {
			t.Errorf("theme.css references %s, which doesn't exist: %v", m[1], err)
		}
	}
}

var (
	hexColourRe  = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b`)
	colourFuncRe = regexp.MustCompile(`(?i)\b(?:rgba?|hsla?|hwb|lab|lch|oklab|oklch|color|color-mix)\(`)
	stripCallsRe = regexp.MustCompile(`(?i)(?:var|url)\([^)]*\)`)
	wordRe       = regexp.MustCompile(`[a-zA-Z]+`)
)

// colourProps are the properties whose value can be a colour.
func isColourProp(prop string) bool {
	if strings.HasPrefix(prop, "--") {
		return false
	}
	for _, p := range []string{"color", "background", "border", "outline", "box-shadow", "text-shadow", "fill", "stroke", "text-decoration", "caret-color", "accent-color", "column-rule"} {
		if prop == p || strings.HasPrefix(prop, p+"-") || strings.HasSuffix(prop, "-"+p) {
			return true
		}
	}
	return false
}

// namedColours are the CSS colour keywords a stylesheet could reach for
// instead of a token. transparent, currentcolor and inherit aren't here:
// they name no colour of their own.
var namedColours = func() map[string]bool {
	m := map[string]bool{}
	for _, n := range strings.Fields(`white black red green blue gray grey silver maroon purple fuchsia lime olive yellow navy teal aqua
		orange pink brown gold cyan magenta beige ivory khaki lavender coral crimson salmon tomato violet indigo turquoise tan
		orchid plum snow linen azure bisque wheat chocolate firebrick goldenrod gainsboro whitesmoke lightgray lightgrey darkgray
		darkgrey dimgray dimgrey slategray slategrey darkred darkblue darkgreen orangered skyblue steelblue royalblue`) {
		m[n] = true
	}
	return m
}()

// SPEC B9.1, B9.9 layer 2: a section stylesheet may only lay things out.
// It declares no colour and no font of its own; both come from theme.css
// tokens.
func TestSectionStylesheetsDeclareNoColourOrFontLiterals(t *testing.T) {
	root := repoRoot(t)
	for _, path := range sectionStylesheets(t) {
		rel, _ := filepath.Rel(root, path)
		css := readCSS(t, path)
		if strings.Contains(css, "@font-face") {
			t.Errorf("%s declares @font-face; fonts are loaded in theme.css only", rel)
		}
		for _, d := range cssDecls(css) {
			where := rel + ` "` + d.selector + `" ` + d.prop
			if isColourProp(d.prop) {
				if hexColourRe.MatchString(d.value) || colourFuncRe.MatchString(d.value) {
					t.Errorf("%s has a colour literal (%s); use a theme.css token", where, d.value)
				}
				for _, w := range wordRe.FindAllString(stripCallsRe.ReplaceAllString(d.value, ""), -1) {
					if namedColours[strings.ToLower(w)] {
						t.Errorf("%s uses the colour name %q (%s); use a theme.css token", where, w, d.value)
					}
				}
			}
			switch d.prop {
			case "font-family":
				if !regexp.MustCompile(`^var\(--font-[a-z]+\)$`).MatchString(d.value) {
					t.Errorf("%s = %s; a font family must be one of the theme.css font tokens", where, d.value)
				}
			case "font":
				if strings.ContainsAny(d.value, `"'`) || !strings.Contains(d.value, "var(--font-") {
					t.Errorf("%s = %s; the font shorthand must take its family from a theme.css font token", where, d.value)
				}
			}
		}
	}
}

// SPEC B9.2 contrast rule, B9.9 layer 4: white on the bright accent is
// 2.85:1 and never allowed; the bright accent is never a text colour.
func TestNoWhiteTextOnAccentAndNoAccentText(t *testing.T) {
	root := repoRoot(t)
	paths := append(sectionStylesheets(t), themeCSSPath(t))
	sort.Strings(paths)
	accent := regexp.MustCompile(`var\(--color-signal-orange\)`)
	white := regexp.MustCompile(`^(var\(--color-white\)|var\(--color-card\)|#fff(fff)?|white)$`)
	for _, path := range paths {
		rel, _ := filepath.Rel(root, path)
		for _, m := range cssBlockRe.FindAllStringSubmatch(readCSS(t, path), -1) {
			selector := strings.TrimSpace(m[1])
			var fill, text string
			for _, d := range cssDecls(m[0]) {
				switch {
				case d.prop == "background" || d.prop == "background-color":
					fill = d.value
				case d.prop == "color":
					text = d.value
				}
			}
			// The one exception: the large HQ of the wordmark is accent on ink
			// navy, the same 6.02:1 pair as ink on accent, and gate 4.13 and
			// the mockup draw it that way. It is only ever set on navy.
			if accent.MatchString(text) && !strings.HasSuffix(selector, ".wordmark strong") {
				t.Errorf("%s %q: text uses --color-signal-orange; use --color-signal-orange-deep", rel, selector)
			}
			if accent.MatchString(fill) && white.MatchString(strings.ToLower(text)) {
				t.Errorf("%s %q: white text on the accent fill is 2.85:1; use --color-ink-navy", rel, selector)
			}
		}
	}
}

// PLAN P4-02: Archivo is retired. Nothing that ships, builds or tests the
// app may mention it (docs and the decisions log describe its history and
// are left alone).
func TestNothingReferencesArchivo(t *testing.T) {
	root := repoRoot(t)
	self := filepath.Join(root, "tools", "policy")
	skipDirs := map[string]bool{"node_modules": true, ".git": true, ".claude": true, "bin": true, "docs": true, "reports": true, "test-results": true}
	needle := regexp.MustCompile(`(?i)archivo`)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] || path == self {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		switch filepath.Ext(path) {
		case ".woff2", ".png", ".jpg", ".gif", ".webp", ".docx", ".db":
		default:
			switch rel {
			case "PLAN.md", "SPEC.md", "PROGRESS.md", "CHANGES.md":
				return nil // planning documents that describe the retirement
			}
			if data, err := os.ReadFile(path); err == nil && needle.Match(data) {
				t.Errorf("%s mentions Archivo, which was retired in v1.1", rel)
			}
		}
		if needle.MatchString(d.Name()) {
			t.Errorf("%s: Archivo files must be deleted", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

var (
	controlTagRe = regexp.MustCompile(`<(button|input|select|textarea)\b[^>]*>`)
	classAttrRe  = regexp.MustCompile(`\sclass="([^"]*)"`)
	typeAttrRe   = regexp.MustCompile(`\stype="([^"]*)"`)
	templateActs = regexp.MustCompile(`\{\{.*?\}\}`)
	cssClassRe   = regexp.MustCompile(`\.([a-zA-Z][\w-]*)`)
)

// SPEC gate 4.04, 4.05 and B9.9 layer 4: no <button>, <select>, <input> or
// <textarea> renders anywhere without a theme class. Buttons are one of
// the three kinds (primary and secondary are .btn, small is .mini) or a
// toolbar or link-style button; every other control is a .field (or a
// .check for a checkbox). A hidden input, and a file chooser that is
// visually hidden behind a real button, don't render.
func TestEveryControlInEveryTemplateHasAThemeClass(t *testing.T) {
	root := repoRoot(t)
	themeClasses := map[string]bool{}
	for _, m := range cssClassRe.FindAllStringSubmatch(readCSS(t, themeCSSPath(t)), -1) {
		themeClasses[m[1]] = true
	}
	allowed := map[string][]string{
		"button":   {"btn", "mini", "tb", "link-button", "icon-btn"},
		"select":   {"field"},
		"textarea": {"field"},
		"input":    {"field", "check", "search-input"},
	}

	checked := 0
	err := filepath.WalkDir(filepath.Join(root, "web", "templates"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, m := range controlTagRe.FindAllStringSubmatch(string(data), -1) {
			tag, kind := m[0], m[1]
			inputType := ""
			if tm := typeAttrRe.FindStringSubmatch(tag); tm != nil {
				inputType = tm[1]
			}
			if kind == "input" && inputType == "hidden" {
				continue
			}
			var classes []string
			if cm := classAttrRe.FindStringSubmatch(tag); cm != nil {
				classes = strings.Fields(templateActs.ReplaceAllString(cm[1], " "))
			}
			checked++
			if kind == "input" && inputType == "file" && slicesContains(classes, "visually-hidden") {
				continue
			}
			ok := false
			for _, c := range classes {
				if slicesContains(allowed[kind], c) && themeClasses[c] {
					ok = true
				}
			}
			if !ok {
				t.Errorf("%s: %s has no theme class (want one of %v from theme.css)", rel, tag, allowed[kind])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 50 {
		t.Errorf("only %d controls were found in the templates; the scan itself is broken", checked)
	}
}

func slicesContains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// expectedIcons is SPEC B9.4's set: the nine new icons, the five section
// icons, and the Word import icon the Knowledge Base editor's button uses.
var expectedIcons = []string{
	"arrow-up", "arrow-down", "trash", "give-back", "plus", "search", "check", "chevron", "close",
	"briefing", "kb", "tasks", "calendar", "settings", "import-word",
}

// SPEC gate 4.12, B9.4: every icon is one drawing on a 24 x 24 grid with a
// 2px round stroke, no fill, and the colour of whatever holds it. None is
// fetched from the internet, and each has a class in theme.css.
func TestIconsAreOneSetOfLineDrawings(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, "web", "static", "theme", "icons")
	css := readCSS(t, themeCSSPath(t))

	for _, name := range expectedIcons {
		data, err := os.ReadFile(filepath.Join(dir, name+".svg"))
		if err != nil {
			t.Errorf("icon %s is missing: %v", name, err)
			continue
		}
		svg := string(data)
		for _, want := range []string{`viewBox="0 0 24 24"`, `fill="none"`, `stroke="currentColor"`, `stroke-width="2"`, `stroke-linecap="round"`, `stroke-linejoin="round"`} {
			if !strings.Contains(svg, want) {
				t.Errorf("icons/%s.svg lacks %s", name, want)
			}
		}
		if regexp.MustCompile(`(?:fill|stroke)="#`).MatchString(svg) {
			t.Errorf("icons/%s.svg hard-codes a colour; icons take the colour of their button", name)
		}
		if !strings.Contains(css, ".icon-"+name+" ") || !strings.Contains(css, "/static/theme/icons/"+name+".svg") {
			t.Errorf("theme.css has no .icon-%s class pointing at %s.svg", name, name)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		// comphq.svg is the app's own mark, not one of the interface's line
		// icons (SPEC B11.2, gate 5.20); it has its own tests in icons_test.go.
		if e.Name() == "comphq.svg" {
			continue
		}
		if strings.HasSuffix(e.Name(), ".svg") && !slicesContains(expectedIcons, strings.TrimSuffix(e.Name(), ".svg")) {
			t.Errorf("icons/%s isn't in the icon set; add it to B9.4's list (and this test) on purpose", e.Name())
		}
	}
}

var (
	buttonElemRe = regexp.MustCompile(`(?s)<button\b([^>]*)>(.*?)</button>`)
	tagRe        = regexp.MustCompile(`(?s)<[^>]*>`)
)

// SPEC gate 4.11: icon-only buttons exist for exactly four things, and each
// has both a title and an aria-label that say the same thing. There is one
// place to write one (the part-icon-button partial); a button anywhere else
// with nothing readable in it is an unnamed icon button and fails.
func TestIconOnlyButtonsAreTheFourPermittedAndFullyNamed(t *testing.T) {
	root := repoRoot(t)
	titleRe := regexp.MustCompile(`\stitle="([^"]*)"`)
	ariaRe := regexp.MustCompile(`\saria-label="([^"]*)"`)

	err := filepath.WalkDir(filepath.Join(root, "web", "templates"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		for _, m := range buttonElemRe.FindAllStringSubmatch(string(data), -1) {
			attrs, inner := m[1], m[2]
			// A template action ({{.Name}}) counts as words: it prints some.
			text := strings.TrimSpace(tagRe.ReplaceAllString(inner, ""))
			isIconBtn := strings.Contains(attrs, "icon-btn")
			if text != "" && !isIconBtn {
				continue // an ordinary button with words
			}
			if rel != "web/templates/app/parts.html" {
				t.Errorf("%s: <button%s> has no words; icon-only buttons are written only in the part-icon-button partial (gate 4.11)", rel, attrs)
				continue
			}
			title, aria := titleRe.FindStringSubmatch(attrs), ariaRe.FindStringSubmatch(attrs)
			if title == nil || aria == nil || title[1] != aria[1] || strings.TrimSpace(title[1]) == "" {
				t.Errorf("%s: the icon-only button needs a title and an aria-label that are the same non-empty text; got %q / %q", rel, title, aria)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
