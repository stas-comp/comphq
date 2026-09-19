package people

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// SPEC gate 4.08: the same person is the same circle colour every time.
func TestAvatarClassIsDeterministic(t *testing.T) {
	if AvatarClass(5, 9) != AvatarClass(5, 9) {
		t.Error("AvatarClass(5, 9) is not deterministic")
	}
}

// Gate 4.08: the person using the app always gets the accent circle, on
// every screen, and nobody else does.
func TestAvatarClassGivesTheCurrentPersonTheAccentCircle(t *testing.T) {
	if got := AvatarClass(7, 7); got != "av me" {
		t.Errorf("AvatarClass(me) = %q, want %q", got, "av me")
	}
	for id := int64(1); id <= 20; id++ {
		if id != 7 && AvatarClass(id, 7) == "av me" {
			t.Errorf("AvatarClass(%d, me=7) = the accent circle, which only person 7 may have", id)
		}
	}
	if got := AvatarClass(0, 0); got == "av me" {
		t.Error("with nobody signed in (meID 0), person 0 must not get the accent circle")
	}
}

// Every class comes from the four-colour palette, whichever id it is
// asked about, including ids that are zero or negative.
func TestAvatarClassIsAlwaysOneOfThePalette(t *testing.T) {
	palette := map[string]bool{}
	for _, c := range avatarClasses {
		palette[c] = true
	}
	seen := map[string]bool{}
	for id := int64(-8); id < 40; id++ {
		class := AvatarClass(id, 0)
		if !palette[class] {
			t.Fatalf("AvatarClass(%d) = %q, not one of %v", id, class, avatarClasses)
		}
		seen[class] = true
	}
	if len(seen) != len(avatarClasses) {
		t.Errorf("only %d of the %d palette colours are ever used", len(seen), len(avatarClasses))
	}
}

// cssRuleBackground reads the background of `selector { ... }` from
// theme.css, resolving one level of var(--token).
func cssRuleBackground(t *testing.T, css, selector string) string {
	t.Helper()
	m := regexp.MustCompile(regexp.QuoteMeta(selector) + `\s*\{[^}]*?background(?:-color)?:\s*([^;}]+)`).FindStringSubmatch(css)
	if m == nil {
		t.Fatalf("theme.css has no %s rule with a background", selector)
	}
	value := strings.TrimSpace(m[1])
	if v := regexp.MustCompile(`^var\((--[a-z0-9-]+)\)$`).FindStringSubmatch(value); v != nil {
		tok := regexp.MustCompile(regexp.QuoteMeta(v[1]) + `:\s*(#[0-9a-fA-F]{6})`).FindStringSubmatch(css)
		if tok == nil {
			t.Fatalf("theme.css doesn't define %s", v[1])
		}
		return tok[1]
	}
	return value
}

func hexLuminance(t *testing.T, hex string) float64 {
	t.Helper()
	if len(hex) != 7 || hex[0] != '#' {
		t.Fatalf("%q is not a #rrggbb colour", hex)
	}
	channel := func(from int) float64 {
		v, err := strconv.ParseUint(hex[from:from+2], 16, 8)
		if err != nil {
			t.Fatal(err)
		}
		return float64(v) / 255
	}
	return relativeLuminance(channel(1), channel(3), channel(5))
}

// TestAvatarPaletteMeetsContrast proves PLAN.md P2-01's "with contrast
// checked" for the v1.1 palette: white initials on each of the four
// circle colours, and ink initials on the accent circle, must meet WCAG
// AA's 4.5:1 — read from the CSS that is actually shipped.
func TestAvatarPaletteMeetsContrast(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "web", "static", "theme", "theme.css"))
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	const minContrast = 4.5

	for _, class := range avatarClasses {
		selector := "." + strings.ReplaceAll(class, " ", ".")
		bg := cssRuleBackground(t, css, selector)
		if got := contrastRatio(1.0, hexLuminance(t, bg)); got < minContrast { // 1.0 = white initials
			t.Errorf("%s: white on %s = %.2f:1, want >= %.1f", selector, bg, got, minContrast)
		}
	}

	me := cssRuleBackground(t, css, "."+strings.ReplaceAll(avatarMeClass, " ", "."))
	ink := hexLuminance(t, "#141b2d")
	if got := contrastRatio(hexLuminance(t, me), ink); got < minContrast {
		t.Errorf(".av.me: ink on %s = %.2f:1, want >= %.1f", me, got, minContrast)
	}
}

// relativeLuminance is the WCAG 2.x formula for sRGB channel values in
// [0,1].
func relativeLuminance(r, g, b float64) float64 {
	lin := func(c float64) float64 {
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// contrastRatio is WCAG 2.x's contrast formula for two relative
// luminances (order doesn't matter — it always divides the lighter by
// the darker).
func contrastRatio(l1, l2 float64) float64 {
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}
