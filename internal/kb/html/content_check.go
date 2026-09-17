package html

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HasUnresolvedContent reports whether bodyHTML still has something
// Settings > Content check should flag (SPEC gates 1.36, 1.49): a
// data-missing-src or data-missing-kind placeholder (an external image
// that couldn't be fetched at publish time, or any unsupported
// Word-import item — a chart, diagram, shape, equation, object, or
// unsupported picture format), or an <img> whose src isn't under
// /images/. Sanitize already prevents that last case on every normal
// path, so it's a defensive check here rather than the expected one.
func HasUnresolvedContent(bodyHTML string) bool {
	found := false
	walk(parseFragment(bodyHTML), func(n *html.Node) {
		if found {
			return
		}
		switch n.DataAtom {
		case atom.Div:
			if _, ok := attrValue(n, "data-missing-kind"); ok {
				found = true
			}
			if _, ok := attrValue(n, "data-missing-src"); ok {
				found = true
			}
		case atom.Img:
			if src, ok := attrValue(n, "src"); !ok || !strings.HasPrefix(src, "/images/") {
				found = true
			}
		}
	})
	return found
}
