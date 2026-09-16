package docx

import (
	"bytes"
	"encoding/xml"
	"io"
)

// paragraphPropsXML is shared by a style definition's own pPr and (later,
// when needed) a paragraph's direct pPr.
type paragraphPropsXML struct {
	OutlineLvl *struct {
		Val int `xml:"val,attr"`
	} `xml:"outlineLvl"`
}

// paragraph is one <w:p>, reduced to what P1-11 needs: enough to find the
// title and headings. P1-29 extends this with runs (marks), lists, links,
// tables and images per SPEC B4.
type paragraph struct {
	styleID          string
	directOutlineLvl *int
	text             string
}

// parseDocument walks document.xml as a token stream rather than modelling
// every element type, so wrapper elements it doesn't know about yet
// (w:sdt, w:smartTag, w:customXml, w:fldSimple, w:hyperlink, text boxes...)
// are simply passed through instead of needing explicit structs.
func parseDocument(data []byte) ([]paragraph, error) {
	if err := checkXMLDepth(data); err != nil {
		if err == errXMLTooDeep {
			return nil, ErrTooLarge
		}
		return nil, ErrUnreadable
	}

	dec := xml.NewDecoder(bytes.NewReader(data))

	var paragraphs []paragraph
	var inParagraph bool
	var cur paragraph
	var buf bytes.Buffer

	// skipDepth counts nested w:del/w:delText/w:moveFrom elements whose
	// text must never appear (SPEC B4: "skip w:del, w:delText and
	// w:moveFrom"). w:ins and w:moveTo are included normally.
	skipDepth := 0

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrUnreadable
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inParagraph = true
				cur = paragraph{}
				buf.Reset()
			case "pStyle":
				if inParagraph {
					cur.styleID = attrVal(t, "val")
				}
			case "outlineLvl":
				if inParagraph {
					v := attrIntVal(t, "val")
					cur.directOutlineLvl = &v
				}
			case "del", "delText", "moveFrom":
				skipDepth++
			case "tab":
				if inParagraph && skipDepth == 0 {
					buf.WriteByte(' ')
				}
			case "br":
				if inParagraph && skipDepth == 0 {
					buf.WriteByte(' ')
				}
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if inParagraph {
					cur.text = buf.String()
					paragraphs = append(paragraphs, cur)
				}
				inParagraph = false
			case "del", "delText", "moveFrom":
				if skipDepth > 0 {
					skipDepth--
				}
			}

		case xml.CharData:
			if inParagraph && skipDepth == 0 {
				buf.Write(t)
			}
		}
	}

	return paragraphs, nil
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
