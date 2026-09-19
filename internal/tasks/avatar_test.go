package tasks

import (
	"testing"

	"github.com/stas-comp/comphq/internal/people"
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

// The Board and the Briefing draw the same person in the same colour.
func TestAvatarClassIsThePeoplePackages(t *testing.T) {
	for id := int64(-3); id < 12; id++ {
		if AvatarClass(id, 5) != people.AvatarClass(id, 5) {
			t.Errorf("AvatarClass(%d, 5) differs from people.AvatarClass", id)
		}
	}
}
