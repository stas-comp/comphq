package html

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// RewriteExternalImages walks rawHTML for <img> elements whose src doesn't
// already point under /images/, calling fetch for each one (SPEC B4:
// "external images on publish"). fetch is a pure indirection — the images
// package (a store with SSRF-safe fetching) supplies it; this function
// knows nothing about HTTP, files or data: decoding, only how to find an
// <img> and how to replace it.
//
// On success (ok=true), the img's src becomes newSrc. On failure, the img
// is replaced with a placeholder div carrying data-missing-src (the
// original URL) — an attribute the Sanitize allowlist already permits —
// and that URL is added to the returned failures list, in document order,
// so the caller can report them (SPEC gate 1.19: "the editor response
// lists the failures").
//
// This must run before Sanitize: afterward, Sanitize unconditionally
// drops any img not already under /images/, which is exactly what should
// happen to whatever this function couldn't rewrite.
func RewriteExternalImages(rawHTML string, fetch func(src string) (newSrc string, ok bool)) (rewritten string, failures []string) {
	nodes := parseFragment(rawHTML)

	rewriteOne := func(n *html.Node) *html.Node {
		src, _ := attrValue(n, "src")
		if src == "" || strings.HasPrefix(src, "/images/") {
			return n
		}
		if newSrc, ok := fetch(src); ok {
			setAttr(n, "src", newSrc)
			return n
		}
		failures = append(failures, src)
		return &html.Node{
			Type:     html.ElementNode,
			Data:     "div",
			DataAtom: atom.Div,
			Attr: []html.Attribute{
				{Key: "data-missing-kind", Val: "external"},
				{Key: "data-missing-src", Val: src},
			},
		}
	}

	// parseFragment's top-level nodes have no .Parent to InsertBefore/
	// RemoveChild against (they're a plain slice relative to a synthetic
	// context node), so a top-level <img> is replaced by rebuilding the
	// slice; a nested one has a real parent and is replaced in the tree.
	var visitChildren func(*html.Node)
	visitChildren = func(n *html.Node) {
		var next *html.Node
		for c := n.FirstChild; c != nil; c = next {
			next = c.NextSibling
			if c.Type == html.ElementNode && c.DataAtom == atom.Img {
				if replacement := rewriteOne(c); replacement != c {
					n.InsertBefore(replacement, c)
					n.RemoveChild(c)
				}
				continue
			}
			visitChildren(c)
		}
	}

	top := make([]*html.Node, len(nodes))
	for i, n := range nodes {
		if n.Type == html.ElementNode && n.DataAtom == atom.Img {
			top[i] = rewriteOne(n)
			continue
		}
		visitChildren(n)
		top[i] = n
	}

	return renderFragment(top), failures
}

func attrValue(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}
