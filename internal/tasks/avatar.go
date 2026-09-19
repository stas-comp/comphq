package tasks

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app/format"
	"github.com/stas-comp/comphq/internal/people"
)

// InitialsFor returns up to two uppercase initials from a person's name
// (SPEC A6: "people's initials in coloured circles"), the same ones the top
// bar's own circle shows.
func InitialsFor(name string) string {
	return format.Initials(name)
}

// AvatarClass returns the CSS classes for a person's initials circle
// (people.AvatarClass: the same colour on every screen, the accent circle
// for the person using the app).
func AvatarClass(personID, meID int64) string {
	return people.AvatarClass(personID, meID)
}

// currentPersonID is the id of the person making the request, or 0.
func currentPersonID(r *http.Request) int64 {
	if p, ok := people.FromContext(r.Context()); ok {
		return p.ID
	}
	return 0
}
