package html

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// blockTags are the elements search and scroll-to address individually
// (SPEC B4): every one gets a data-b id, in document order, including one
// nested inside another block (a list item inside a list item gets its
// own id, separate from its parent's).
var blockTags = map[string]bool{
	"p": true, "h2": true, "h3": true, "li": true,
	"blockquote": true, "tr": true, "figcaption": true,
}

// AssignBlocks numbers every block element in document order with
// data-b="<n>", starting at 1 (SPEC B3: block 0 is reserved for the
// article title, which lives outside body_html).
func AssignBlocks(sanitized string) string {
	next := 1
	nodes := walk(parseFragment(sanitized), func(n *html.Node) {
		if !blockTags[n.Data] {
			return
		}
		setAttr(n, "data-b", strconv.Itoa(next))
		next++
	})
	return renderFragment(nodes)
}

func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

// BlockTexts extracts each block's own plain text, keyed by its data-b id,
// from HTML that AssignBlocks has already numbered. Text belongs to the
// innermost enclosing block: a list item's text doesn't include a nested
// list's text, since the nested items are blocks of their own.
func BlockTexts(withBlockIDs string) map[int]string {
	texts := make(map[int]string)
	var stack []int

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			if len(stack) > 0 {
				texts[stack[len(stack)-1]] += n.Data
			}
		case html.ElementNode:
			if id, ok := blockID(n); ok {
				stack = append(stack, id)
				defer func() { stack = stack[:len(stack)-1] }()
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	for _, n := range parseFragment(withBlockIDs) {
		visit(n)
	}

	for id, text := range texts {
		texts[id] = strings.Join(strings.Fields(text), " ")
	}
	return texts
}

// AddHighlightData sets a data-hl attribute on each block named in
// highlights (keyed by data-b id) to its given HTML string, for
// highlight.js to swap in client-side (SPEC B4: "matched words come from
// the server via highlight() on that block, passed as a data attribute").
// It's a display-only transform — the result is never what gets saved.
func AddHighlightData(bodyHTML string, highlights map[int]string) string {
	if len(highlights) == 0 {
		return bodyHTML
	}
	nodes := walk(parseFragment(bodyHTML), func(n *html.Node) {
		id, ok := blockID(n)
		if !ok {
			return
		}
		if hl, found := highlights[id]; found {
			setAttr(n, "data-hl", hl)
		}
	})
	return renderFragment(nodes)
}

func blockID(n *html.Node) (int, bool) {
	for _, a := range n.Attr {
		if a.Key == "data-b" {
			id, err := strconv.Atoi(a.Val)
			return id, err == nil
		}
	}
	return 0, false
}
