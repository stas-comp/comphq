package html

import "testing"

func TestAddHighlightData(t *testing.T) {
	in := `<p data-b="1">Ask Sam about the toner.</p><p data-b="2">Wipe the glass.</p>`
	got := AddHighlightData(in, map[int]string{1: "Ask Sam about the <mark>toner</mark>."})

	want := `<p data-b="1" data-hl="Ask Sam about the &lt;mark&gt;toner&lt;/mark&gt;.">Ask Sam about the toner.</p><p data-b="2">Wipe the glass.</p>`
	if got != want {
		t.Errorf("AddHighlightData =\n%q\nwant\n%q", got, want)
	}
}

func TestAddHighlightDataNoMatchesLeavesBodyUnchanged(t *testing.T) {
	in := `<p data-b="1">Nothing matched.</p>`
	if got := AddHighlightData(in, nil); got != in {
		t.Errorf("AddHighlightData(nil) = %q, want unchanged %q", got, in)
	}
}
