package tasks

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestInitialsFor(t *testing.T) {
	cases := []struct{ name, want string }{
		{"Sam", "S"},
		{"Sam Jones", "SJ"},
		{"sam", "S"},
		{"  Sam   Jones  ", "SJ"},
		{"Sam Middle Jones", "SJ"},
		{"", ""},
	}
	for _, c := range cases {
		if got := InitialsFor(c.name); got != c.want {
			t.Errorf("InitialsFor(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestAvatarColorClassIsDeterministic(t *testing.T) {
	if AvatarColorClass(5) != AvatarColorClass(5) {
		t.Error("AvatarColorClass(5) is not deterministic")
	}
}

func TestAvatarColorClassIsAlwaysOneOfTheDefinedClasses(t *testing.T) {
	re := regexp.MustCompile(`^task-avatar-(\d+)$`)
	for id := int64(0); id < int64(avatarColorCount)*3; id++ {
		class := AvatarColorClass(id)
		m := re.FindStringSubmatch(class)
		if m == nil {
			t.Fatalf("AvatarColorClass(%d) = %q, doesn't match task-avatar-N", id, class)
		}
		n, _ := strconv.Atoi(m[1])
		if n < 0 || n >= avatarColorCount {
			t.Fatalf("AvatarColorClass(%d) = %q, index out of range [0,%d)", id, class, avatarColorCount)
		}
	}
}

// TestAvatarColorClassCSSMatchesGoPalette guards against the CSS
// palette (web/static/tasks/board.css, hand-written since a class
// selector can't be templated at request time — SPEC's CSP has no
// 'unsafe-inline' for style-src) drifting from AvatarColorClass's own
// notion of how many entries exist and what hue each one is.
func TestAvatarColorClassCSSMatchesGoPalette(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "web", "static", "tasks", "board.css"))
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)

	rule := regexp.MustCompile(`\.task-avatar-(\d+)\s*\{\s*background-color:\s*hsl\((\d+),\s*(\d+)%,\s*(\d+)%\);`)
	matches := rule.FindAllStringSubmatch(css, -1)
	if len(matches) != avatarColorCount {
		t.Fatalf("board.css has %d .task-avatar-N rules, want %d (avatarColorCount)", len(matches), avatarColorCount)
	}

	seen := make(map[int]bool)
	for _, m := range matches {
		index, _ := strconv.Atoi(m[1])
		hue, _ := strconv.Atoi(m[2])
		sat, _ := strconv.Atoi(m[3])
		light, _ := strconv.Atoi(m[4])
		seen[index] = true

		if want := avatarHue(index); hue != want {
			t.Errorf("task-avatar-%d hue = %d, want %d (avatarHue(%d))", index, hue, want, index)
		}
		if sat != avatarSaturation {
			t.Errorf("task-avatar-%d saturation = %d%%, want %d%%", index, sat, avatarSaturation)
		}
		if light != avatarLightness {
			t.Errorf("task-avatar-%d lightness = %d%%, want %d%%", index, light, avatarLightness)
		}
	}
	for i := 0; i < avatarColorCount; i++ {
		if !seen[i] {
			t.Errorf("board.css is missing a .task-avatar-%d rule", i)
		}
	}
}

// TestAvatarColorMeetsContrastForEveryHue proves PLAN.md P2-01's "a
// colour derived from the person id, with contrast checked": white text
// on the avatar background must meet WCAG AA's 4.5:1 contrast ratio at
// every possible hue (not just the ones the fixed palette currently
// uses), since avatarSaturation/avatarLightness are fixed constants
// applied to every hue alike — so the palette could grow later without
// ever needing this check revisited.
func TestAvatarColorMeetsContrastForEveryHue(t *testing.T) {
	const minContrast = 4.5
	for hue := 0; hue < 360; hue++ {
		r, g, b := hslToRGB(float64(hue), float64(avatarSaturation)/100, float64(avatarLightness)/100)
		contrast := contrastRatio(1.0, relativeLuminance(r, g, b)) // 1.0 = white text
		if contrast < minContrast {
			t.Fatalf("hue %d: contrast against white = %.2f, want >= %.1f (rgb=%.3f,%.3f,%.3f)", hue, contrast, minContrast, r, g, b)
		}
	}
}

// hslToRGB converts HSL (h in [0,360), s and l in [0,1]) to linear-scale
// sRGB channel values in [0,1], following the standard algorithm.
func hslToRGB(h, s, l float64) (r, g, b float64) {
	c := (1 - math.Abs(2*l-1)) * s
	hp := h / 60
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	var r1, g1, b1 float64
	switch {
	case hp < 1:
		r1, g1, b1 = c, x, 0
	case hp < 2:
		r1, g1, b1 = x, c, 0
	case hp < 3:
		r1, g1, b1 = 0, c, x
	case hp < 4:
		r1, g1, b1 = 0, x, c
	case hp < 5:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}
	m := l - c/2
	return r1 + m, g1 + m, b1 + m
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
