package html

import (
	"strings"
	"testing"
)

func TestOpeningWords(t *testing.T) {
	long := strings.Repeat("alpha beta ", 30) // 330 characters
	cases := []struct {
		name, in, want string
	}{
		{"first paragraph, headings skipped", `<h2>Opening hours</h2><p>We are open <b>nine</b> to five.</p><p>Second.</p>`, "We are open nine to five."},
		{"a list item stands in when there is no paragraph", `<h3>Steps</h3><ul><li>Unlock the door</li><li>Lights</li></ul>`, "Unlock the door"},
		{"an outer list item before its nested one", `<ul><li>outer<ul><li>inner</li></ul></li></ul>`, "outer"},
		{"numbered blocks read the same", `<h2 data-b="1">H</h2><p data-b="2">Body text</p>`, "Body text"},
		{"only headings", `<h2>Just a heading</h2>`, ""},
		{"empty", ``, ""},
		{"blank paragraphs are skipped", `<p>   </p><p>Real words</p>`, "Real words"},
		{"cut at a word boundary", "<p>" + long + "</p>", strings.TrimSpace(strings.Repeat("alpha beta ", 12)) + " alpha…"},
	}
	for _, c := range cases {
		got := OpeningWords(c.in, 140)
		if got != c.want {
			t.Errorf("%s: OpeningWords(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
	// Never longer than the limit plus the ellipsis, and never mid-word.
	got := OpeningWords("<p>"+long+"</p>", 140)
	if n := len([]rune(got)); n > 141 {
		t.Errorf("cut text has %d characters", n)
	}
	// One word with no spaces is cut where it stands.
	if got := OpeningWords("<p>"+strings.Repeat("x", 300)+"</p>", 140); got != strings.Repeat("x", 140)+"…" {
		t.Errorf("a single long word: got %d characters", len([]rune(got)))
	}
	// Accented text is counted in characters, not bytes.
	if got := OpeningWords("<p>"+strings.Repeat("é", 100)+"</p>", 140); got != strings.Repeat("é", 100) {
		t.Errorf("accents were cut early: %q", got)
	}
}
