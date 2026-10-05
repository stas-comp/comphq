package html

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// OpeningWords is the sentence a category's page shows under an article's
// title (SPEC gate 7.72, B13.10): the plain text of the first block that is
// not a heading, cut at a word boundary to about max characters, with "…"
// when it was cut. It reads the stored body, whether or not its blocks have
// been numbered, and returns "" for a body with no text.
func OpeningWords(body string, max int) string {
	text := firstBodyText(body)
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	runes := []rune(text)
	cut := max
	for cut > 0 && runes[cut] != ' ' { // runes[cut] is the first rune left out: a space means the cut is already clean
		cut--
	}
	if cut == 0 { // one very long word: cut it where it stands
		cut = max
	}
	return strings.TrimRight(string(runes[:cut]), " .,;:!?-–—") + "…"
}

// firstBodyText finds the first non-heading block, in document order, that
// has text of its own. A list item's text does not include a nested list's
// (those items are blocks of their own), the same rule BlockTexts uses.
func firstBodyText(body string) string {
	type frame struct {
		tag  string
		text strings.Builder
	}
	var stack, opened []*frame

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			if len(stack) > 0 {
				stack[len(stack)-1].text.WriteString(n.Data)
			}
		case html.ElementNode:
			if blockTags[n.Data] {
				f := &frame{tag: n.Data}
				stack, opened = append(stack, f), append(opened, f)
				defer func() { stack = stack[:len(stack)-1] }()
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	for _, n := range parseFragment(body) {
		visit(n)
	}
	for _, f := range opened {
		if f.tag == "h2" || f.tag == "h3" {
			continue
		}
		if text := strings.Join(strings.Fields(f.text.String()), " "); text != "" {
			return text
		}
	}
	return ""
}
