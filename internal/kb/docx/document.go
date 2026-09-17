package docx

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// paragraphPropsXML is shared by a style definition's own pPr and (used
// directly, below) a paragraph's direct pPr.
type paragraphPropsXML struct {
	OutlineLvl *struct {
		Val int `xml:"val,attr"`
	} `xml:"outlineLvl"`
}

// onOffXML models a WordprocessingML on/off toggle element (<w:b/>,
// <w:i/>): present but with no w:val, or w:val="1"/"true", means on;
// w:val="0"/"false" means off (SPEC B4: "honouring w:val=0/false").
type onOffXML struct {
	Val string `xml:"val,attr"`
}

func (o onOffXML) bool() bool {
	switch strings.ToLower(o.Val) {
	case "0", "false":
		return false
	default:
		return true
	}
}

// runPropsXML is a style definition's own top-level <w:rPr> (the default
// run formatting for text in a paragraph using that style — distinct from
// a paragraph's own <w:pPr><w:rPr> for its paragraph-mark run, which SPEC
// B4 doesn't ask this converter to read).
type runPropsXML struct {
	Bold   *onOffXML `xml:"b"`
	Italic *onOffXML `xml:"i"`
}

type segmentKind int

const (
	segText segmentKind = iota
	segBreak
)

// segment is one contiguous run of a paragraph's content: either text
// with its resolved bold/italic, or a line break (SPEC B4: "w:br -> br").
type segment struct {
	kind   segmentKind
	text   string
	bold   bool
	italic bool
}

// paragraph is one <w:p>.
type paragraph struct {
	styleID          string
	directOutlineLvl *int
	segments         []segment
}

func (p paragraph) plainText() string {
	var b strings.Builder
	for _, s := range p.segments {
		if s.kind == segText {
			b.WriteString(s.text)
		}
	}
	return b.String()
}

// parsedDocument is parseDocument's result: the paragraphs, plus one note
// per comment, footnote or endnote reference found (SPEC B4: "left out,
// and counted in notes" — headers and footers, being separate parts
// rather than inline references, are added by the caller instead).
type parsedDocument struct {
	paragraphs []paragraph
	notes      []string
}

// parseDocument walks document.xml as a token stream rather than
// modelling every element type, so wrapper elements that need no special
// handling (w:sdt/w:sdtContent, w:smartTag, w:customXml, w:fldSimple,
// w:hyperlink, VML text boxes...) are simply passed through: the token
// stream doesn't care about ancestors this walker isn't matching on, so
// a <w:p> nested inside any of them is still found by its own start/end
// tokens (SPEC B4: "descend into w:sdt/w:sdtContent, w:smartTag,
// w:customXml and w:fldSimple").
func parseDocument(data []byte, styles styleSheet) (parsedDocument, error) {
	if err := checkXMLDepth(data); err != nil {
		if err == errXMLTooDeep {
			return parsedDocument{}, ErrTooLarge
		}
		return parsedDocument{}, ErrUnreadable
	}

	dec := xml.NewDecoder(bytes.NewReader(data))

	var out parsedDocument
	var inParagraph bool
	var cur paragraph

	var inRun bool
	var inRunProps bool
	var runStyleID string
	var runBold, runItalic *bool
	var runText bytes.Buffer

	flushRun := func() {
		if runText.Len() > 0 {
			bold, italic := styles.resolveMarks(cur.styleID, runStyleID, runBold, runItalic)
			cur.segments = append(cur.segments, segment{kind: segText, text: runText.String(), bold: bold, italic: italic})
			runText.Reset()
		}
	}

	// skipDepth counts nested elements whose text must never appear (SPEC
	// B4: "skip w:del, w:delText and w:moveFrom"; "skip field instruction
	// text"). w:ins and w:moveTo need no equivalent entry — their text is
	// included exactly like any other run's, simply by not being skipped.
	skipDepth := 0

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return parsedDocument{}, ErrUnreadable
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inParagraph = true
				cur = paragraph{}
			case "pStyle":
				if inParagraph {
					cur.styleID = attrVal(t, "val")
				}
			case "outlineLvl":
				if inParagraph {
					v := attrIntVal(t, "val")
					cur.directOutlineLvl = &v
				}
			case "r":
				// A run outside any paragraph isn't valid WordprocessingML
				// (every w:r belongs to some w:p), but staying defensive
				// here means a stray one is simply ignored rather than
				// silently lost into a paragraph struct that was already
				// appended and will never be updated again.
				if inParagraph {
					inRun = true
					runStyleID = ""
					runBold = nil
					runItalic = nil
					runText.Reset()
				}
			case "rPr":
				inRunProps = true
			case "rStyle":
				if inRunProps {
					runStyleID = attrVal(t, "val")
				}
			case "b":
				if inRunProps {
					v := onOffXML{Val: attrVal(t, "val")}.bool()
					runBold = &v
				}
			case "i":
				if inRunProps {
					v := onOffXML{Val: attrVal(t, "val")}.bool()
					runItalic = &v
				}
			case "del", "delText", "moveFrom", "instrText":
				skipDepth++
			case "tab":
				if inRun && skipDepth == 0 {
					runText.WriteByte(' ')
				}
			case "br":
				if inRun && skipDepth == 0 {
					// Page and column breaks carry no visible content in
					// an article body (SPEC B4: "ignoring page and column
					// breaks"); a plain line break (no type, or the
					// explicit default "textWrapping") becomes a real <br>.
					if breakType := attrVal(t, "type"); breakType != "page" && breakType != "column" {
						flushRun()
						cur.segments = append(cur.segments, segment{kind: segBreak})
					}
				}
			case "commentReference", "footnoteReference", "endnoteReference":
				out.notes = append(out.notes, noteFor(t.Name.Local))
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if inParagraph {
					flushRun()
					out.paragraphs = append(out.paragraphs, cur)
				}
				inParagraph = false
			case "r":
				flushRun()
				inRun = false
			case "rPr":
				inRunProps = false
			case "del", "delText", "moveFrom", "instrText":
				if skipDepth > 0 {
					skipDepth--
				}
			}

		case xml.CharData:
			if inRun && skipDepth == 0 {
				runText.Write(t)
			}
		}
	}

	return out, nil
}

func noteFor(elementName string) string {
	switch elementName {
	case "commentReference":
		return "a comment"
	case "footnoteReference":
		return "a footnote"
	case "endnoteReference":
		return "an endnote"
	default:
		return elementName
	}
}

func attrVal(t xml.StartElement, local string) string {
	for _, a := range t.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func attrIntVal(t xml.StartElement, local string) int {
	v := attrVal(t, local)
	n := 0
	neg := false
	for i, c := range v {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n
}
