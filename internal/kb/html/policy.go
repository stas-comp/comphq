// Package html is the single server-side HTML policy for article bodies
// (SPEC B4): Sanitize strips everything not on the allowlist, AssignBlocks
// numbers the block elements search and scroll-to rely on, and BlockTexts
// extracts each block's plain text for the search index.
package html

import "github.com/microcosm-cc/bluemonday"

// policy is the SPEC B4 structural allowlist: a fixed set of tags, table
// cell spans, and the two "missing content" placeholder attributes. No
// style, class, font or id ever survives. Link scheme and image src
// restrictions run as a separate pass below: bluemonday's own URL
// validation is all-or-nothing across href/src together (and rejects
// relative URLs, which /images/... paths are), so it can't express "keep
// the <a> but drop a javascript: href" and "drop the whole <img> unless
// its src starts with /images/" at the same time.
var policy = newPolicy()

func newPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements(
		"p", "h2", "h3", "strong", "em", "ul", "ol", "li", "a",
		"table", "thead", "tbody", "tr", "th", "td", "br", "img",
		"figure", "figcaption", "blockquote", "div",
	)
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("colspan", "rowspan").OnElements("td", "th")
	p.AllowAttrs("src", "alt").OnElements("img")
	p.AllowAttrs("data-missing-kind", "data-missing-src").OnElements("div")
	return p
}
