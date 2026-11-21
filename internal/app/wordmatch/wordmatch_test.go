package wordmatch

import (
	"reflect"
	"testing"
)

func TestWords(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"mailing lists", []string{"mailing", "lists"}},
		{"  Électricité,  du bureau! ", []string{"Électricité", "du", "bureau"}},
		{"a", nil},
		{"", nil},
		{"x1", []string{"x1"}},
		{"a b", []string{"a", "b"}},
	} {
		if got := Words(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Words(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestQueryMatchesEveryWordCaseAndAccentFoldedWithTheLastWordHalfTyped(t *testing.T) {
	q := Parse("électricité bur")
	for _, text := range []string{"ÉLECTRICITÉ du bureau", "le bureau et l'électricité"} {
		if !q.MatchesAll(text) {
			t.Errorf("%q should match", text)
		}
	}
	for _, text := range []string{"électricité du siège", "bureau seulement", ""} {
		if q.MatchesAll(text) {
			t.Errorf("%q should not match", text)
		}
	}
	if Parse("a").MatchesAll("anything") || !Parse("a").Empty() {
		t.Errorf("a one-letter query finds nothing")
	}
	if !Parse("mail").StartsWithFirst("Mailing SOP") || Parse("mail").StartsWithFirst("Printing the mailing") || Parse("").StartsWithFirst("x") {
		t.Errorf("StartsWithFirst")
	}
}

func TestIndexAndLower(t *testing.T) {
	if got := Index(Lower("Hello World"), Lower("WORLD")); got != 6 {
		t.Errorf("Index = %d", got)
	}
	if Index(Lower("abc"), nil) != -1 || Index(Lower("ab"), Lower("abc")) != -1 {
		t.Errorf("Index of nothing or too long")
	}
	if len(Lower("ÉÀ")) != 2 {
		t.Errorf("Lower keeps one rune per rune")
	}
}
