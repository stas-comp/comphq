package tasks

import (
	"fmt"
	"strings"
)

// avatarSaturation and avatarLightness are fixed so that white text is
// always readable on the resulting colour, whichever hue a palette entry
// uses (verified against the worst case — yellow, which maximises both
// the red and green channels' WCAG weights most heavily — by
// TestAvatarColorMeetsContrastForEveryHue).
const (
	avatarSaturation = 50
	avatarLightness  = 25
)

// avatarColorCount is the number of pre-defined .task-avatar-N CSS
// classes in web/static/tasks/board.css, one per palette entry (evenly
// spaced around the hue wheel). A fixed class name, not an inline style,
// because the app's CSP has no 'unsafe-inline' for style-src (or
// default-src, which style-src falls back to) — an inline
// background-color is silently dropped by the browser, never applied.
const avatarColorCount = 12

// InitialsFor returns up to two uppercase initials from a person's name
// (SPEC A6: "people's initials in coloured circles").
func InitialsFor(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	initials := strings.ToUpper(firstRune(fields[0]))
	if len(fields) > 1 {
		initials += strings.ToUpper(firstRune(fields[len(fields)-1]))
	}
	return initials
}

func firstRune(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

// AvatarColorClass returns the CSS class for a person's initials circle,
// deterministically derived from their id (PLAN.md P2-01: "a colour
// derived from the person id, with contrast checked") — the same person
// always gets the same colour, and white text on it always meets WCAG AA
// contrast (>= 4.5:1, see TestAvatarColorMeetsContrastForEveryHue).
func AvatarColorClass(personID int64) string {
	// 5 is coprime with avatarColorCount (12), so consecutive ids cycle
	// through every entry before repeating, instead of stepping by 1 and
	// landing on adjacent (similar-looking) hues for neighbouring ids.
	index := ((personID*5)%avatarColorCount + avatarColorCount) % avatarColorCount
	return fmt.Sprintf("task-avatar-%d", index)
}

// avatarHue is the hue (degrees) for palette entry i of avatarColorCount,
// evenly spaced — used only by the CSS generator comment and the test
// that verifies every entry meets contrast; the actual CSS values in
// board.css are written out statically, not generated at build time.
func avatarHue(i int) int {
	return i * 360 / avatarColorCount
}
