package docx

import (
	"html"
	"strconv"
	"strings"
)

// buildResult determines the title (SPEC B4: "Title, or a first non-empty
// paragraph in heading 1, becomes the article title and is removed from
// the body. Otherwise the title is the file name without its extension.")
// and renders the remaining blocks. Images are added in P1-32.
func buildResult(blocks []block, notes []string, styles styleSheet, numbering numberingSheet, filename string) Result {
	title, body := extractTitle(blocks, styles, filename)

	var htmlBuilder strings.Builder
	i := 0
	for i < len(body) {
		if body[i].kind == blockTable {
			htmlBuilder.WriteString(renderTable(body[i].table, styles, numbering))
			i++
			continue
		}

		// A run of consecutive paragraph blocks, so list grouping (which
		// needs to see neighbouring items) works the same regardless of
		// what comes after it — a table ends a list exactly like an
		// ordinary interrupting paragraph does, simply by not being one.
		j := i
		var paragraphs []paragraph
		for j < len(body) && body[j].kind == blockParagraph {
			paragraphs = append(paragraphs, body[j].paragraph)
			j++
		}
		htmlBuilder.WriteString(renderParagraphSequence(paragraphs, styles, numbering))
		i = j
	}

	return Result{Title: title, HTML: htmlBuilder.String(), Notes: notes}
}

// renderParagraphSequence renders a run of paragraphs — the top-level
// body between tables, or one table cell's own content — dropping empty
// paragraphs (SPEC B4) and grouping consecutive same-numId list items
// into nested ul/ol structures before rendering everything else as
// headings or plain paragraphs.
func renderParagraphSequence(paragraphs []paragraph, styles styleSheet, numbering numberingSheet) string {
	// Filtered up front, before list grouping, so a blank paragraph
	// between two list items (common visual spacing in Word, not a real
	// interruption) doesn't split them into two separate lists.
	nonEmpty := make([]paragraph, 0, len(paragraphs))
	for _, p := range paragraphs {
		if p.hasContent() {
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
	return htmlBuilder.String()
}

// positionedCell is one row's cell together with its 0-indexed starting
// grid column, computed from the colSpans of the cells before it in the
// same row (including a vMerge "continue" cell, which still occupies a
// real column even though it renders nothing of its own).
type positionedCell struct {
	startCol int
	cell     tableCell
}

func positionRow(row tableRow) []positionedCell {
	col := 0
	out := make([]positionedCell, 0, len(row.cells))
	for _, c := range row.cells {
		out = append(out, positionedCell{startCol: col, cell: c})
		col += c.colSpan
	}
	return out
}

// renderTable renders a table (SPEC B4: "w:tbl -> table", "w:tblHeader
// rows -> th", "w:gridSpan -> colspan", "w:vMerge restart/continue ->
// rowspan computed per column"). A vMerge "continue" cell is dropped
// entirely; a "restart" cell's rowspan is however many further rows have
// a "continue" cell at that same starting column.
func renderTable(tbl tableBlock, styles styleSheet, numbering numberingSheet) string {
	positioned := make([][]positionedCell, len(tbl.rows))
	for i, row := range tbl.rows {
		positioned[i] = positionRow(row)
	}

	var b strings.Builder
	b.WriteString("<table>\n")
	for i, row := range tbl.rows {
		b.WriteString("<tr>\n")
		cellTag := "td"
		if row.header {
			cellTag = "th"
		}
		for _, pc := range positioned[i] {
			if pc.cell.vMergeKind == "continue" {
				continue
			}

			rowspan := 1
			if pc.cell.vMergeKind == "restart" {
				for k := i + 1; k < len(tbl.rows); k++ {
					continues := false
					for _, pc2 := range positioned[k] {
						if pc2.startCol == pc.startCol && pc2.cell.vMergeKind == "continue" {
							continues = true
							break
						}
					}
					if !continues {
						break
					}
					rowspan++
				}
			}

			b.WriteString("<" + cellTag)
			if pc.cell.colSpan > 1 {
				b.WriteString(` colspan="` + strconv.Itoa(pc.cell.colSpan) + `"`)
			}
			if rowspan > 1 {
				b.WriteString(` rowspan="` + strconv.Itoa(rowspan) + `"`)
			}
			b.WriteString(">")
			b.WriteString(renderParagraphSequence(pc.cell.paragraphs, styles, numbering))
			b.WriteString("</" + cellTag + ">\n")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</table>\n")
	return b.String()
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

// extractTitle looks only at paragraph blocks — a title can't come from
// inside a table (SPEC B4 doesn't say otherwise, and a table's own first
// cell being silently promoted to the document title would be surprising).
func extractTitle(blocks []block, styles styleSheet, filename string) (string, []block) {
	for i, b := range blocks {
		if b.kind != blockParagraph {
			continue
		}
		p := b.paragraph
		if styles.isTitleStyle(p.styleID) && strings.TrimSpace(p.plainText()) != "" {
			return strings.TrimSpace(p.plainText()), removeBlockAt(blocks, i)
		}
	}
	for i, b := range blocks {
		if b.kind != blockParagraph {
			continue
		}
		p := b.paragraph
		tag, isHeading := styles.headingLevel(p.styleID, p.directOutlineLvl)
		if isHeading && tag == "h2" && strings.TrimSpace(p.plainText()) != "" {
			return strings.TrimSpace(p.plainText()), removeBlockAt(blocks, i)
		}
	}
	return titleFromFilename(filename), blocks
}

func removeBlockAt(blocks []block, i int) []block {
	out := make([]block, 0, len(blocks)-1)
	out = append(out, blocks[:i]...)
	out = append(out, blocks[i+1:]...)
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
		if seg.kind == segMedia {
			writeMediaSegment(&b, seg)
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

// writeMediaSegment renders a real image or a data-missing-kind
// placeholder (SPEC B4). A placeholder's own explanatory text comes from
// the editor's CSS, keyed off data-missing-kind's value (P1-33), the
// same way an existing missing-picture placeholder from the paste flow
// already works.
func writeMediaSegment(b *strings.Builder, seg segment) {
	if seg.missingKind != "" {
		b.WriteString(`<div data-missing-kind="`)
		b.WriteString(html.EscapeString(seg.missingKind))
		b.WriteString(`"></div>`)
		return
	}
	b.WriteString(`<img src="`)
	b.WriteString(html.EscapeString(seg.imageToken))
	b.WriteString(`" alt="`)
	b.WriteString(html.EscapeString(seg.imageAlt))
	b.WriteString(`">`)
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
// break or a piece of media rather than reaching past it.
func trimSegments(segs []segment) []segment {
	out := make([]segment, len(segs))
	copy(out, segs)
	for i := range out {
		if out[i].kind != segText {
			break
		}
		out[i].text = strings.TrimLeft(out[i].text, " \t")
		if out[i].text != "" {
			break
		}
	}
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].kind != segText {
			break
		}
		out[i].text = strings.TrimRight(out[i].text, " \t")
		if out[i].text != "" {
			break
		}
	}
	return out
}
