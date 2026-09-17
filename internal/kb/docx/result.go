package docx

import (
	"html"
	"strings"
)

// buildResult determines the title (SPEC B4: "Title, or a first non-empty
// paragraph in heading 1, becomes the article title and is removed from
// the body. Otherwise the title is the file name without its extension.")
// and renders the remaining paragraphs, grouping consecutive list items
// into nested ul/ol structures. Tables and images are added in P1-31–P1-32.
func buildResult(paragraphs []paragraph, notes []string, styles styleSheet, numbering numberingSheet, filename string) Result {
	title, body := extractTitle(paragraphs, styles, filename)

	// SPEC B4: "Drop empty paragraphs." Filtered up front, before list
	// grouping, so a blank paragraph between two list items (common
	// visual spacing in Word, not a real interruption) doesn't split them
	// into two separate lists.
	nonEmpty := make([]paragraph, 0, len(body))
	for _, p := range body {
		if strings.TrimSpace(p.plainText()) != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}

	var htmlBuilder strings.Builder
	i := 0
	for i < len(nonEmpty) {
		p := nonEmpty[i]

		if numID, _, isList := effectiveListInfo(p, styles); isList {
			j := i
			var items []paragraph
			var levels []int
			for j < len(nonEmpty) {
				n2, l2, ok2 := effectiveListInfo(nonEmpty[j], styles)
				// SPEC B4: "Consecutive paragraphs with the same numId
				// form one list" — anything else, including a list
				// paragraph with a *different* numId, ends this one.
				if !ok2 || n2 != numID {
					break
				}
				items = append(items, nonEmpty[j])
				levels = append(levels, l2)
				j++
			}
			htmlBuilder.WriteString(renderList(items, levels, numbering, numID))
			i = j
			continue
		}

		tag, isHeading := styles.headingLevel(p.styleID, p.directOutlineLvl)
		if !isHeading {
			tag = "p"
		}
		htmlBuilder.WriteString(renderParagraphHTML(p, tag, isHeading))
		i++
	}

	return Result{Title: title, HTML: htmlBuilder.String(), Notes: notes}
}

// effectiveListInfo resolves a paragraph's list membership: direct
// w:numPr wins if present, otherwise its paragraph style's (SPEC B4).
// numId 0 is never a list, from either source.
func effectiveListInfo(p paragraph, styles styleSheet) (numID, ilvl int, isList bool) {
	if p.directNumID != nil {
		numID = *p.directNumID
		if p.directIlvl != nil {
			ilvl = *p.directIlvl
		}
		return numID, ilvl, numID != 0
	}
	if n, l, ok := styles.numbering(p.styleID); ok {
		return n, l, n != 0
	}
	return 0, 0, false
}

// listStackEntry is one open (still unclosed) ul/ol in renderList's
// nesting stack.
type listStackEntry struct {
	ilvl int
	tag  string
}

// renderList turns a run of consecutive same-numId list paragraphs into a
// nested ul/ol structure (SPEC B4: "ilvl -> nesting"). A jump to a deeper
// ilvl opens a new list inside the current, still-open <li>; a jump back
// out closes lists until the stack matches (or is shallower than) the new
// item's level, then opens one more level if needed — level changes in
// real documents are usually by one step, but this handles an arbitrary
// jump the same way.
func renderList(items []paragraph, levels []int, numbering numberingSheet, numID int) string {
	var b strings.Builder
	var stack []listStackEntry

	for idx, p := range items {
		ilvl := levels[idx]
		tag := "ol"
		if numbering.isBullet(numID, ilvl) {
			tag = "ul"
		}

		for len(stack) > 0 && stack[len(stack)-1].ilvl > ilvl {
			top := stack[len(stack)-1]
			b.WriteString("</li></" + top.tag + ">\n")
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 || stack[len(stack)-1].ilvl < ilvl {
			b.WriteString("<" + tag + ">\n")
			stack = append(stack, listStackEntry{ilvl: ilvl, tag: tag})
		} else {
			b.WriteString("</li>\n")
		}
		b.WriteString("<li>")
		b.WriteString(renderInlineSegments(p.segments, false))
	}

	for len(stack) > 0 {
		top := stack[len(stack)-1]
		b.WriteString("</li></" + top.tag + ">\n")
		stack = stack[:len(stack)-1]
	}
	return b.String()
}

func extractTitle(paragraphs []paragraph, styles styleSheet, filename string) (string, []paragraph) {
	for i, p := range paragraphs {
		if styles.isTitleStyle(p.styleID) && strings.TrimSpace(p.plainText()) != "" {
			return strings.TrimSpace(p.plainText()), removeAt(paragraphs, i)
		}
	}
	for i, p := range paragraphs {
		tag, isHeading := styles.headingLevel(p.styleID, p.directOutlineLvl)
		if isHeading && tag == "h2" && strings.TrimSpace(p.plainText()) != "" {
			return strings.TrimSpace(p.plainText()), removeAt(paragraphs, i)
		}
	}
	return titleFromFilename(filename), paragraphs
}

func removeAt(paragraphs []paragraph, i int) []paragraph {
	out := make([]paragraph, 0, len(paragraphs)-1)
	out = append(out, paragraphs[:i]...)
	out = append(out, paragraphs[i+1:]...)
	return out
}

func titleFromFilename(filename string) string {
	name := filename
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' || name[i] == '\\' {
			name = name[i+1:]
			break
		}
	}
	if dot := strings.LastIndexByte(name, '.'); dot > 0 {
		name = name[:dot]
	}
	return name
}

// renderParagraphHTML renders one paragraph's segments as tag's content
// (SPEC B4). Bold is suppressed inside headings ("No strong inside
// headings"); list items render the same inline content via
// renderInlineSegments directly, with no paragraph tag of their own.
func renderParagraphHTML(p paragraph, tag string, suppressStrong bool) string {
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(tag)
	b.WriteString(">")
	b.WriteString(renderInlineSegments(p.segments, suppressStrong))
	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">\n")
	return b.String()
}

// renderInlineSegments renders a run of segments: consecutive text
// segments sharing the same href are grouped into one <a> first (so a
// link spanning several differently-formatted runs becomes one anchor,
// not several adjacent ones — SPEC B4's "complex-field hyperlink
// spanning runs"), then within each such group (or outside any link),
// consecutive segments with identical bold/italic are merged into a
// single marked-up span — a run split mid-word, for instance, must read
// as one continuous word, not two adjacent but visually identical
// elements. Line breaks become a literal <br>.
func renderInlineSegments(segments []segment, suppressStrong bool) string {
	segs := trimSegments(segments)

	var b strings.Builder
	i := 0
	for i < len(segs) {
		seg := segs[i]
		if seg.kind == segBreak {
			b.WriteString("<br>")
			i++
			continue
		}
		if seg.text == "" {
			i++
			continue
		}
		j := i
		for j < len(segs) && segs[j].kind == segText && segs[j].href == seg.href {
			j++
		}
		if seg.href != "" {
			b.WriteString(`<a href="`)
			b.WriteString(html.EscapeString(seg.href))
			b.WriteString(`">`)
			writeMarkedSegments(&b, segs[i:j], suppressStrong)
			b.WriteString(`</a>`)
		} else {
			writeMarkedSegments(&b, segs[i:j], suppressStrong)
		}
		i = j
	}
	return b.String()
}

func writeMarkedSegments(b *strings.Builder, segs []segment, suppressStrong bool) {
	i := 0
	for i < len(segs) {
		seg := segs[i]
		if seg.text == "" {
			i++
			continue
		}
		bold := seg.bold && !suppressStrong
		italic := seg.italic
		text := seg.text
		j := i + 1
		for j < len(segs) && segs[j].bold == seg.bold && segs[j].italic == seg.italic {
			text += segs[j].text
			j++
		}
		writeMarkedText(b, text, bold, italic)
		i = j
	}
}

func writeMarkedText(b *strings.Builder, text string, bold, italic bool) {
	open, close := "", ""
	switch {
	case bold && italic:
		open, close = "<strong><em>", "</em></strong>"
	case bold:
		open, close = "<strong>", "</strong>"
	case italic:
		open, close = "<em>", "</em>"
	}
	b.WriteString(open)
	b.WriteString(html.EscapeString(text))
	b.WriteString(close)
}

// trimSegments trims leading whitespace from the first text segment and
// trailing whitespace from the last, without touching which formatting
// applies to the visible characters in between, and stops at a line
// break rather than reaching past it.
func trimSegments(segs []segment) []segment {
	out := make([]segment, len(segs))
	copy(out, segs)
	for i := range out {
		if out[i].kind == segBreak {
			break
		}
		out[i].text = strings.TrimLeft(out[i].text, " \t")
		if out[i].text != "" {
			break
		}
	}
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].kind == segBreak {
			break
		}
		out[i].text = strings.TrimRight(out[i].text, " \t")
		if out[i].text != "" {
			break
		}
	}
	return out
}
