package docx

import (
	"bytes"
	"encoding/xml"
)

// Pictures in every kind of shape (SPEC B13.9 items 2-4, gates 7.61, 7.65):
// a drawing is walked for every picture in it, in reading order, instead of
// being read as "one picture or a shape"; a VML picture can hold several; and
// an mc:AlternateContent takes the branch that shows more pictures.

// pictureRef is one picture found inside a drawing: the relationship that
// holds its bytes, and its alt text.
type pictureRef struct {
	relID string
	alt   string
}

// drawingContent is what scanDrawing finds in a <w:drawing>: every picture
// (a:blip r:embed anywhere inside, including a group's, a canvas's and a
// shape's picture fill), and, when there are none, what kind of thing it is.
type drawingContent struct {
	pictures []pictureRef
	kind     string // "chart", "diagram" or "shape", when there are no pictures
}

// scanDrawing walks the inside of a <w:drawing> (its inline/anchor and
// everything below). It deliberately does not descend into a:extLst's
// svgBlip: an SVG icon's PNG blip is the one read (SPEC B4).
func scanDrawing(inner string) drawingContent {
	var out drawingContent
	uri := ""
	docAlt := ""
	picAlt := ""
	dec := xml.NewDecoder(bytes.NewReader(wrapInnerXML(inner)))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "docPr":
			if docAlt == "" {
				docAlt = attrVal(se, "descr")
				if docAlt == "" {
					docAlt = attrVal(se, "title")
				}
			}
		case "pic":
			picAlt = ""
		case "cNvPr":
			picAlt = attrVal(se, "descr")
			if picAlt == "" {
				picAlt = attrVal(se, "title")
			}
		case "graphicData":
			if uri == "" {
				uri = attrVal(se, "uri")
			}
		case "blip":
			if id := attrVal(se, "embed"); id != "" {
				alt := picAlt
				if alt == "" {
					alt = docAlt
				}
				out.pictures = append(out.pictures, pictureRef{relID: id, alt: alt})
			}
		}
	}
	if len(out.pictures) == 0 {
		switch {
		case hasSuffixFold(uri, "/chart"):
			out.kind = "chart"
		case hasSuffixFold(uri, "/diagram"):
			out.kind = "diagram"
		default:
			out.kind = "shape"
		}
	}
	return out
}

// resolveDrawingSegments turns a <w:drawing>'s content into segments: one
// picture each, in order, or a single placeholder when it holds no picture.
func (ctx *docCtx) resolveDrawingSegments(inner string) []segment {
	content := scanDrawing(inner)
	if len(content.pictures) == 0 {
		return []segment{{kind: segMedia, missingKind: content.kind}}
	}
	segs := make([]segment, 0, len(content.pictures))
	for _, p := range content.pictures {
		segs = append(segs, ctx.resolvePictureSegment(p.relID, p.alt))
	}
	return segs
}

// vmlItem is one thing found in a VML <w:pict> or <w:object>, in order: a
// picture (v:imagedata r:id, or a shape filled with a picture, v:fill r:id) or
// a text box's own inner XML.
type vmlItem struct {
	imageRelID string
	txbxInner  string
}

// scanVML finds every picture and every text box in a VML subtree, in
// reading order, wherever they sit in it: a v:group can hold several shapes,
// and v:shape, v:rect, v:roundrect... all vary by drawing tool.
func scanVML(data []byte) []vmlItem {
	var out []vmlItem
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "imagedata", "fill":
			if id := attrVal(se, "id"); id != "" {
				out = append(out, vmlItem{imageRelID: id})
			}
		case "txbxContent":
			var raw struct {
				InnerXML string `xml:",innerxml"`
			}
			if err := dec.DecodeElement(&raw, &se); err == nil {
				out = append(out, vmlItem{txbxInner: raw.InnerXML})
			}
		}
	}
	return out
}

// countPictureRefs counts the picture references in some XML: a:blip r:embed,
// v:imagedata r:id and v:fill r:id. It is how an mc:AlternateContent's two
// branches are compared, and how a header's pictures are counted.
func countPictureRefs(data []byte) int {
	n := 0
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "blip":
			if attrVal(se, "embed") != "" {
				n++
			}
		case "imagedata", "fill":
			if attrVal(se, "id") != "" {
				n++
			}
		}
	}
	return n
}

// chooseAlternate picks the branch of an mc:AlternateContent to use (B13.9
// item 3, SPEC B4: "the one branch that yields an image or text"). This
// converter understands none of the extensions a Choice requires, so the
// Fallback is the safe default (it is also what avoids counting a picture or
// text box twice when both branches show it); but when the Choice shows more
// pictures than the Fallback, the Choice wins, and with no Fallback at all the
// Choice is all there is. It returns the chosen branch's inner XML, or "".
func chooseAlternate(inner string) string {
	var choice, fallback string
	hasFallback := false
	dec := xml.NewDecoder(bytes.NewReader(wrapInnerXML(inner)))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		var raw struct {
			InnerXML string `xml:",innerxml"`
		}
		switch se.Name.Local {
		case "Choice":
			if err := dec.DecodeElement(&raw, &se); err == nil && choice == "" {
				choice = raw.InnerXML
			}
		case "Fallback":
			hasFallback = true
			if err := dec.DecodeElement(&raw, &se); err == nil && fallback == "" {
				fallback = raw.InnerXML
			}
		}
	}
	switch {
	case countPictureRefs(wrapInnerXML(choice)) > countPictureRefs(wrapInnerXML(fallback)):
		return choice
	case hasFallback:
		return fallback
	default:
		return choice
	}
}
