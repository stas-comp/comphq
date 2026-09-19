package tasks

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app/format"
	"github.com/stas-comp/comphq/internal/people"
)

// avatarClasses are the mockup's four circle colours (SPEC gate 4.08,
// web/static/theme/theme.css `.av`, `.av.b`, `.av.c`, `.av.d`): the default
// ink navy and three quiet blues, browns and greens, every one dark
// enough for white initials (TestAvatarPaletteMeetsContrast). A fixed
// class, not an inline style: the app's CSP has no 'unsafe-inline' for
// style-src (or default-src, which style-src falls back to), so an inline
// background-color is silently dropped rather than applied.
var avatarClasses = []string{"av", "av b", "av c", "av d"}

// avatarMeClass is the accent circle the person using the app always gets
// (gate 4.08), ink initials on the accent fill.
const avatarMeClass = "av me"

// InitialsFor returns up to two uppercase initials from a person's name
// (SPEC A6: "people's initials in coloured circles"), the same ones the top
// bar's own circle shows.
func InitialsFor(name string) string {
	return format.Initials(name)
}

// AvatarClass returns the CSS classes for a person's initials circle: the
// accent circle when they are the person using the app (meID), otherwise
// one of the four palette colours picked from their id, so the same
// person is the same colour on every screen (gate 4.08). meID is 0 when
// nobody is signed in, which is nobody's id.
func AvatarClass(personID, meID int64) string {
	if meID != 0 && personID == meID {
		return avatarMeClass
	}
	n := int64(len(avatarClasses))
	return avatarClasses[((personID%n)+n)%n]
}

// currentPersonID is the id of the person making the request, or 0.
func currentPersonID(r *http.Request) int64 {
	if p, ok := people.FromContext(r.Context()); ok {
		return p.ID
	}
	return 0
}
