package html

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// RewriteImageSrcs rewrites every <img> src via toRelative — used by
// Export everything (SPEC B5) to turn the live /images/<hash>.<ext>
// route into a path relative to a self-contained exported page. It
// returns the original srcs too, in document order without duplicates,
// so the caller knows which files to copy alongside the page.
func RewriteImageSrcs(bodyHTML string, toRelative func(src string) string) (rewritten string, srcs []string) {
	seen := map[string]bool{}
	nodes := walk(parseFragment(bodyHTML), func(n *html.Node) {
		if n.DataAtom != atom.Img {
			return
		}
		src, ok := attrValue(n, "src")
		if !ok {
			return
		}
		if !seen[src] {
			seen[src] = true
			srcs = append(srcs, src)
		}
		setAttr(n, "src", toRelative(src))
	})
	return renderFragment(nodes), srcs
}
