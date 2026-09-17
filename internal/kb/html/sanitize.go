package html

import (
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var imgSrcPolicy = regexp.MustCompile(`^/images/`)

var allowedLinkSchemes = map[string]bool{"http": true, "https": true, "mailto": true}

// Sanitize is the one place article bodies get made safe (SPEC B4): remap
// headings to the two sizes the editor offers, strip everything the
// structural policy doesn't allow, restrict links to http/https/mailto
// (adding rel="noopener" to every one that's left), and drop any image
// whose src isn't already under /images/. Re-sanitising already-sanitised
// HTML gives the same output back.
func Sanitize(raw string) string {
	remapped := renderFragment(walk(parseFragment(raw), remapHeading))
	sanitized := policy.Sanitize(remapped)
	nodes := walk(parseFragment(sanitized), fixLink)
	nodes = pruneInvalidImages(nodes)
	return renderFragment(nodes)
}

// parseFragment parses an HTML fragment as it would appear inside <body>,
// matching how bluemonday itself parses (and how the block-id pass below
// re-parses its output).
func parseFragment(s string) []*html.Node {
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(s), body)
	if err != nil {
		return nil
	}
	return nodes
}

func renderFragment(nodes []*html.Node) string {
	var buf strings.Builder
	for _, n := range nodes {
		html.Render(&buf, n)
	}
	return buf.String()
}

// walk applies fn to every element node in the fragment, in document
// order, then returns the same node list (fn mutates nodes in place).
func walk(nodes []*html.Node, fn func(*html.Node)) []*html.Node {
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			fn(n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	for _, n := range nodes {
		visit(n)
	}
	return nodes
}

// remapHeading maps h1 to h2, and h4-h6 to h3, before sanitising (SPEC B4:
// the editor only ever offers two heading sizes).
func remapHeading(n *html.Node) {
	switch n.Data {
	case "h1":
		n.Data = "h2"
		n.DataAtom = atom.H2
	case "h4", "h5", "h6":
		n.Data = "h3"
		n.DataAtom = atom.H3
	}
}

// fixLink keeps a link's href only if it parses with an allowed scheme
// (SPEC B4: "limited to http, https and mailto"), then adds rel="noopener"
// to every <a> regardless — the tag itself is always kept; only the href
// is conditional.
func fixLink(n *html.Node) {
	if n.DataAtom != atom.A {
		return
	}
	kept := n.Attr[:0]
	for _, a := range n.Attr {
		if a.Key != "href" {
			continue
		}
		if u, err := url.Parse(a.Val); err == nil && allowedLinkSchemes[strings.ToLower(u.Scheme)] {
			kept = append(kept, a)
		}
	}
	n.Attr = append(kept, html.Attribute{Key: "rel", Val: "noopener"})
}

// pruneInvalidImages removes every <img> whose src doesn't already start
// with /images/ (SPEC B4) — uploads and the Word importer rewrite to that
// prefix before Sanitize ever runs, so anything else (an external URL, a
// javascript: URL, or a missing src) is dropped outright rather than left
// as a broken image.
func pruneInvalidImages(nodes []*html.Node) []*html.Node {
	kept := nodes[:0]
	for _, n := range nodes {
		pruneInvalidImageChildren(n)
		if n.Type == html.ElementNode && n.DataAtom == atom.Img && !hasValidImageSrc(n) {
			continue
		}
		kept = append(kept, n)
	}
	return kept
}

func pruneInvalidImageChildren(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		pruneInvalidImageChildren(c)
		if c.Type == html.ElementNode && c.DataAtom == atom.Img && !hasValidImageSrc(c) {
			n.RemoveChild(c)
		}
	}
}

func hasValidImageSrc(n *html.Node) bool {
	src, ok := attrValue(n, "src")
	return ok && imgSrcPolicy.MatchString(src)
}
