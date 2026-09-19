package people

// A person is a coloured circle holding their initials, the same colour on
// every screen (SPEC gate 4.08). The palette is the mockup's four circle
// colours (web/static/theme/theme.css `.av`, `.av.b`, `.av.c`, `.av.d`), each
// dark enough for white initials (TestAvatarPaletteMeetsContrast). A fixed
// class, not an inline style: the app's CSP has no 'unsafe-inline' for
// style-src (or default-src, which style-src falls back to), so an inline
// background-color is silently dropped rather than applied.
var avatarClasses = []string{"av", "av b", "av c", "av d"}

// avatarMeClass is the accent circle the person using the app always gets
// (ink initials on the accent fill; white on it is banned, D-60).
const avatarMeClass = "av me"

// AvatarClass returns the CSS classes for a person's initials circle: the
// accent circle when they are the person using the app (meID), otherwise
// one of the four palette colours picked from their id. meID is 0 when
// nobody is signed in, which is nobody's id.
func AvatarClass(personID, meID int64) string {
	if meID != 0 && personID == meID {
		return avatarMeClass
	}
	n := int64(len(avatarClasses))
	return avatarClasses[((personID%n)+n)%n]
}
