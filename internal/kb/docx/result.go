package docx

import (
	"html"
	"strings"
)

// buildResult determines the title (SPEC B4: "Title, or a first non-empty
// paragraph in heading 1, becomes the article title and is removed from
// the body. Otherwise the title is the file name without its extension.")
// and renders the remaining paragraphs. Lists, links, tables and images
// are added in P1-30–P1-32.
func buildResult(paragraphs []paragraph, notes []string, styles styleSheet, filename string) Result {
	title, body := extractTitle(paragraphs, styles, filename)

	var htmlBuilder strings.Builder
	for _, p := range body {
		if strings.TrimSpace(p.plainText()) == "" {
			continue // SPEC B4: "Drop empty paragraphs."
		}
		tag, isHeading := styles.headingLevel(p.styleID, p.directOutlineLvl)
		if !isHeading {
			tag = "p"
		}
		htmlBuilder.WriteString(renderParagraphHTML(p, tag, isHeading))
	}

	return Result{Title: title, HTML: htmlBuilder.String(), Notes: notes}
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

// renderParagraphHTML renders one paragraph's segments as tag's content:
// consecutive text segments with identical resolved formatting are
// merged into a single marked-up span first (a run split mid-word, for
// instance, must read as one continuous word, not two adjacent but
// visually identical elements), line breaks become a literal <br>, and
// bold is suppressed inside headings (SPEC B4: "No strong inside
// headings").
func renderParagraphHTML(p paragraph, tag string, suppressStrong bool) string {
	segs := trimSegments(p.segments)

	var b strings.Builder
	b.WriteString("<")
	b.WriteString(tag)
	b.WriteString(">")

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
		bold := seg.bold && !suppressStrong
		italic := seg.italic
		text := seg.text
		j := i + 1
		for j < len(segs) && segs[j].kind == segText && segs[j].bold == seg.bold && segs[j].italic == seg.italic {
			text += segs[j].text
			j++
		}
		writeMarkedText(&b, text, bold, italic)
		i = j
	}

	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">\n")
	return b.String()
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
