package docx

import (
	"html"
	"strings"
)

// buildResult determines the title (SPEC B4: "Title, or a first non-empty
// paragraph in heading 1, becomes the article title and is removed from
// the body. Otherwise the title is the file name without its extension.")
// and renders the remaining paragraphs. Run-level formatting, lists,
// links, tables and images are added in P1-29–P1-33; for now every
// paragraph is plain escaped text.
func buildResult(paragraphs []paragraph, styles styleSheet, filename string) Result {
	title, body := extractTitle(paragraphs, styles, filename)

	var htmlBuilder strings.Builder
	for _, p := range body {
		text := strings.TrimSpace(p.text)
		if text == "" {
			continue // SPEC B4: "Drop empty paragraphs."
		}
		tag, isHeading := styles.headingLevel(p.styleID, p.directOutlineLvl)
		if !isHeading {
			tag = "p"
		}
		htmlBuilder.WriteString("<")
		htmlBuilder.WriteString(tag)
		htmlBuilder.WriteString(">")
		htmlBuilder.WriteString(html.EscapeString(text))
		htmlBuilder.WriteString("</")
		htmlBuilder.WriteString(tag)
		htmlBuilder.WriteString(">\n")
	}

	return Result{Title: title, HTML: htmlBuilder.String()}
}

func extractTitle(paragraphs []paragraph, styles styleSheet, filename string) (string, []paragraph) {
	for i, p := range paragraphs {
		if styles.isTitleStyle(p.styleID) && strings.TrimSpace(p.text) != "" {
			return strings.TrimSpace(p.text), removeAt(paragraphs, i)
		}
	}
	for i, p := range paragraphs {
		tag, isHeading := styles.headingLevel(p.styleID, p.directOutlineLvl)
		if isHeading && tag == "h2" && strings.TrimSpace(p.text) != "" {
			return strings.TrimSpace(p.text), removeAt(paragraphs, i)
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
