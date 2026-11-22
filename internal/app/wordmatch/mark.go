package wordmatch

import (
	"html"
	"html/template"
	"sort"
	"strings"
	"unicode"
)

// Marking the words that matched, shared by every search that shows a passage (job
// search, Questions): the text is escaped first and the matched words are wrapped
// in <mark>, so nothing typed can inject markup.

// PassageRunes is about how long a passage is.
const PassageRunes = 120

// span is a half-open range of rune indexes.
type span struct{ from, to int }

// matchSpans finds every occurrence of every word in text, merged.
func matchSpans(text []rune, words [][]rune) []span {
	low := make([]rune, len(text))
	for i, r := range text {
		low[i] = unicode.ToLower(r)
	}
	var spans []span
	for _, w := range words {
		for start := 0; start < len(low); {
			i := Index(low[start:], w)
			if i < 0 {
				break
			}
			spans = append(spans, span{start + i, start + i + len(w)})
			start += i + len(w)
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].from < spans[j].from })
	var merged []span
	for _, sp := range spans {
		if n := len(merged); n > 0 && sp.from <= merged[n-1].to {
			if sp.to > merged[n-1].to {
				merged[n-1].to = sp.to
			}
			continue
		}
		merged = append(merged, sp)
	}
	return merged
}

// markRange renders runes [from, to) of text as HTML: escaped first, then the
// matched words wrapped in <mark>. Escaping before marking means a title can
// never inject markup.
func markRange(text []rune, spans []span, from, to int) template.HTML {
	var b strings.Builder
	pos := from
	for _, sp := range spans {
		if sp.to <= from || sp.from >= to {
			continue
		}
		s, e := max(sp.from, from), min(sp.to, to)
		b.WriteString(html.EscapeString(string(text[pos:s])))
		b.WriteString("<mark>")
		b.WriteString(html.EscapeString(string(text[s:e])))
		b.WriteString("</mark>")
		pos = e
	}
	b.WriteString(html.EscapeString(string(text[pos:to])))
	return template.HTML(b.String()) //nolint:gosec // every piece above is html.EscapeString'd
}

// Highlight is text, escaped, with every matched word marked.
func Highlight(text string, words [][]rune) template.HTML {
	r := []rune(text)
	return markRange(r, matchSpans(r, words), 0, len(r))
}

// Passage is about PassageRunes of the notes around the first match, with
// the words marked, or "" when no word is in the notes. Newlines become spaces.
func Passage(notes string, words [][]rune) template.HTML {
	if notes == "" {
		return ""
	}
	text := []rune(strings.Join(strings.Fields(notes), " "))
	spans := matchSpans(text, words)
	if len(spans) == 0 {
		return ""
	}
	from := max(0, spans[0].from-PassageRunes/3)
	to := min(len(text), from+PassageRunes)
	// Start and end on word boundaries, so a passage doesn't open mid-word.
	for from > 0 && from < spans[0].from && text[from-1] != ' ' {
		from++
	}
	body := markRange(text, spans, from, to)
	out := string(body)
	if from > 0 {
		out = "…" + out
	}
	if to < len(text) {
		out += "…"
	}
	return template.HTML(out) //nolint:gosec // built from markRange's escaped output
}
