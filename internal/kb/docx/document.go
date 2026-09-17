package docx

import (
	"bytes"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
)

// paragraphPropsXML is a style definition's own pPr (SPEC B4: a paragraph
// style can set default numbering for every paragraph using it, "a list
// defined via paragraph style").
type paragraphPropsXML struct {
	OutlineLvl *struct {
		Val int `xml:"val,attr"`
	} `xml:"outlineLvl"`
	NumPr *numPrXML `xml:"numPr"`
}

type numPrXML struct {
	Ilvl *struct {
		Val int `xml:"val,attr"`
	} `xml:"ilvl"`
	NumID *struct {
		Val int `xml:"val,attr"`
	} `xml:"numId"`
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
	segMedia
)

// segment is one contiguous unit of a paragraph's content: text with its
// resolved bold/italic and, if inside a link, its href; a line break
// (SPEC B4: "w:br -> br"); or a piece of media — either a successfully
// extracted image (imageToken/imageAlt set) or a placeholder for
// something that couldn't come across (missingKind set, one of
// "picture"/"chart"/"diagram"/"shape"/"equation"/"object" — SPEC B4).
type segment struct {
	kind        segmentKind
	text        string
	bold        bool
	italic      bool
	href        string
	imageToken  string
	imageAlt    string
	missingKind string
}

// paragraph is one <w:p>. directNumID/directIlvl are nil when the
// paragraph has no direct w:numPr — it may still be a list item through
// its paragraph style (SPEC B4), resolved later against styleSheet.
type paragraph struct {
	styleID          string
	directOutlineLvl *int
	directNumID      *int
	directIlvl       *int
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

// hasContent reports whether p has anything worth keeping: real text, or
// a piece of media (an image or a data-missing-kind placeholder). Used
// instead of plainText() alone to decide whether to drop an empty
// paragraph (SPEC B4) — a paragraph holding only a picture has no text
// at all, but is exactly the kind of paragraph that rule shouldn't drop.
func (p paragraph) hasContent() bool {
	if strings.TrimSpace(p.plainText()) != "" {
		return true
	}
	for _, s := range p.segments {
		if s.kind == segMedia {
			return true
		}
	}
	return false
}

type blockKind int

const (
	blockParagraph blockKind = iota
	blockTable
)

// block is one body-level (or table-cell-level) item: a paragraph or a
// table (SPEC B4).
type block struct {
	kind      blockKind
	paragraph paragraph
	table     tableBlock
}

// parsedDocument is parseBlocks's result: the blocks, plus one note per
// comment, footnote, endnote reference, or unsupported item found (SPEC
// B4: "left out, and counted in notes" — headers and footers, being
// separate parts rather than inline references, are added by the caller
// instead).
type parsedDocument struct {
	blocks []block
	notes  []string
}

// docCtx carries what a body-level parse needs but doesn't want to keep
// re-deriving or re-threading as separate parameters: style/hyperlink
// lookups, and the running image list and token counter a single
// Convert() call accumulates across every recursive re-parse (a table
// cell's content, a text box's). Passed by pointer so a table cell or
// text box's own recursive parseBlocks call shares the same running
// state as the top-level one, rather than starting fresh.
type docCtx struct {
	styles         styleSheet
	hyperlinkRels  map[string]string
	resolvedImages map[string]resolvedImage
	images         []Image
	imageCounter   int
}

// resolveDrawingSegment turns a decoded <w:drawing> into a segment: a
// real image (SPEC B4: "w:drawing ... a:blip r:embed"), or a placeholder
// for a chart, SmartArt diagram, or anything else this converter doesn't
// specifically recognise (rendered as a generic "shape" placeholder,
// covering real drawn shapes and any other DrawingML content).
func (ctx *docCtx) resolveDrawingSegment(raw drawingXML) segment {
	body := raw.Inline
	if body == nil {
		body = raw.Anchor
	}
	if body == nil {
		return segment{kind: segMedia, missingKind: "object"}
	}
	alt := body.DocPr.Descr
	if alt == "" {
		alt = body.DocPr.Title
	}
	switch {
	case hasSuffixFold(body.GraphicData.URI, "/picture"):
		relID := ""
		if body.GraphicData.Pic != nil {
			relID = body.GraphicData.Pic.BlipFill.Blip.Embed
		}
		return ctx.resolvePictureSegment(relID, alt)
	case hasSuffixFold(body.GraphicData.URI, "/chart"):
		return segment{kind: segMedia, missingKind: "chart"}
	case hasSuffixFold(body.GraphicData.URI, "/diagram"):
		return segment{kind: segMedia, missingKind: "diagram"}
	default:
		return segment{kind: segMedia, missingKind: "shape"}
	}
}

// resolvePictureSegment looks relID up in the images already resolved
// (read and sniffed) by docx.go's Convert before parsing ever started.
// Anything not found there — an unrecognised id, or one that resolved to
// an unsupported format, an external target, or an oversize file — is a
// "picture" placeholder, never included as image bytes (SPEC B4: "never
// fetched").
func (ctx *docCtx) resolvePictureSegment(relID, alt string) segment {
	resolved, ok := ctx.resolvedImages[relID]
	if !ok || !resolved.supported {
		return segment{kind: segMedia, missingKind: "picture"}
	}
	ctx.imageCounter++
	token := "cid:docximage" + strconv.Itoa(ctx.imageCounter)
	ctx.images = append(ctx.images, Image{Token: token, Bytes: resolved.bytes, Alt: alt})
	return segment{kind: segMedia, imageToken: token, imageAlt: alt}
}

func noteForMissingKind(kind string) string {
	switch kind {
	case "picture":
		return "a picture"
	case "chart":
		return "a chart"
	case "diagram":
		return "a SmartArt diagram"
	case "equation":
		return "an equation"
	case "object":
		return "an embedded object"
	default:
		return "a shape"
	}
}

// vmlContent is what scanVMLPict finds inside a <w:pict> (or <w:object>):
// at most one of an image relationship id or a text box's own inner XML,
// found anywhere in the subtree regardless of which VML shape element
// wraps it (v:shape, v:rect, v:roundrect... all vary by drawing tool, so
// this looks for the content that actually matters rather than modelling
// every shape element).
type vmlContent struct {
	imageRelID string
	txbxInner  string
}

func scanVMLPict(data []byte) vmlContent {
	var out vmlContent
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
		case "imagedata":
			out.imageRelID = attrVal(se, "id")
		case "txbxContent":
			var raw struct {
				InnerXML string `xml:",innerxml"`
			}
			if err := dec.DecodeElement(&raw, &se); err == nil {
				out.txbxInner = raw.InnerXML
			}
		}
	}
	return out
}

// parseBlocks walks a WordprocessingML body (document.xml, or a table
// cell's or text box's own captured fragment, re-parsed recursively) as a
// token stream rather than modelling every element type, so wrapper
// elements that need no special handling (w:sdt/w:sdtContent, w:smartTag,
// w:customXml...) are simply passed through: the token stream doesn't
// care about ancestors this walker isn't matching on, so a <w:p> nested
// inside any of them is still found by its own start/end tokens (SPEC B4:
// "descend into w:sdt/w:sdtContent, w:smartTag, w:customXml and
// w:fldSimple").
//
// A <w:tbl> is decoded whole, via encoding/xml's struct-based
// DecodeElement rather than more token-matching, since a table's row/cell
// nesting is naturally tree-shaped; each cell's own content is captured
// as raw XML and fed back into this same function, so a cell (or a
// table nested inside one, which SPEC B4 flattens into that cell's own
// paragraphs) gets exactly the same marks/links/list handling as any
// other paragraph. A <w:txbxContent> (a text box's paragraphs) is
// likewise decoded whole and queued to be appended right after the
// paragraph that anchors it (SPEC B4), rather than processed inline,
// since a text box's own <w:p> would otherwise be encountered while the
// anchor's <w:p> is still open — this walker has no nesting counter for
// w:p, by design, so a genuinely nested one needs to be routed around it,
// not through it. <w:drawing>, <w:pict>, <w:object> and <m:oMath> are
// decoded (or scanned) the same way, becoming either a real image or a
// data-missing-kind placeholder segment.
func parseBlocks(data []byte, ctx *docCtx) (parsedDocument, error) {
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

	// hrefStack holds the currently-open link's href, from either a
	// w:hyperlink with an external relationship or a HYPERLINK field's
	// resolved URL (SPEC B4). An internal-anchor-only w:hyperlink, or a
	// field that isn't a recognised external link, pushes "" — its
	// content still flows through normally, just with no link (SPEC B4:
	// "internal anchors keep their text and drop the link").
	var hrefStack []string

	// Complex-field state (w:fldChar begin/separate/end plus the
	// w:instrText between begin and separate): fieldState tracks which
	// phase, if any, is open; fieldInstrBuf accumulates the field's own
	// instruction text (which may span more than one w:instrText/run) so
	// it can be parsed once separate is reached, without ever letting it
	// reach the visible run text.
	const (
		fieldIdle = iota
		fieldCollectingInstr
		fieldInResult
	)
	fieldState := fieldIdle
	var fieldInstrBuf bytes.Buffer
	fieldPushedHref := false
	var inInstrText bool
	fldSimplePushedHref := false

	// pendingTextBoxParagraphs holds a text box's already-parsed
	// paragraphs (SPEC B4: "output their paragraphs after the paragraph
	// that anchors them") until the currently-open anchor paragraph closes.
	var pendingTextBoxParagraphs []paragraph

	flushRun := func() {
		if runText.Len() > 0 {
			bold, italic := ctx.styles.resolveMarks(cur.styleID, runStyleID, runBold, runItalic)
			href := ""
			if len(hrefStack) > 0 {
				href = hrefStack[len(hrefStack)-1]
			}
			cur.segments = append(cur.segments, segment{kind: segText, text: runText.String(), bold: bold, italic: italic, href: href})
			runText.Reset()
		}
	}

	appendMedia := func(seg segment) {
		cur.segments = append(cur.segments, seg)
		if seg.missingKind != "" {
			out.notes = append(out.notes, noteForMissingKind(seg.missingKind))
		}
	}

	// skipDepth counts nested elements whose text must never appear in the
	// visible run text (SPEC B4: "skip w:del, w:delText and w:moveFrom";
	// "skip field instruction text"), and, for mc:Choice, whose *content*
	// (including any table/text box/picture inside it) must never be
	// used at all. w:ins and w:moveTo need no equivalent entry — their
	// text is included exactly like any other run's, simply by not being
	// skipped.
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
			case "numId":
				if inParagraph {
					v := attrIntVal(t, "val")
					cur.directNumID = &v
				}
			case "ilvl":
				if inParagraph {
					v := attrIntVal(t, "val")
					cur.directIlvl = &v
				}
			case "hyperlink":
				href := ""
				if relID := attrVal(t, "id"); relID != "" {
					href = ctx.hyperlinkRels[relID]
				}
				hrefStack = append(hrefStack, href)
			case "fldSimple":
				if url, ok := parseHyperlinkFieldInstr(attrVal(t, "instr")); ok {
					hrefStack = append(hrefStack, url)
					fldSimplePushedHref = true
				} else {
					fldSimplePushedHref = false
				}
			case "fldChar":
				switch attrVal(t, "fldCharType") {
				case "begin":
					fieldState = fieldCollectingInstr
					fieldInstrBuf.Reset()
					fieldPushedHref = false
				case "separate":
					if fieldState == fieldCollectingInstr {
						if url, ok := parseHyperlinkFieldInstr(fieldInstrBuf.String()); ok {
							hrefStack = append(hrefStack, url)
							fieldPushedHref = true
						}
					}
					fieldState = fieldInResult
				case "end":
					if fieldState == fieldInResult && fieldPushedHref {
						hrefStack = hrefStack[:len(hrefStack)-1]
					}
					fieldState = fieldIdle
					fieldPushedHref = false
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
			case "del", "delText", "moveFrom", "instrText", "Choice":
				// Choice is mc:Choice: SPEC B4 says AlternateContent must
				// use exactly one branch, and this converter understands
				// none of the extensions a Choice branch Requires, so it
				// always prefers mc:Fallback — skipping Choice's content
				// the same way deleted/field-instruction text is skipped
				// keeps that simple, without a second mechanism.
				skipDepth++
				if t.Name.Local == "instrText" {
					inInstrText = true
				}
			case "tbl":
				// Always consumed via DecodeElement so the decoder's
				// position ends up past the whole element either way, but
				// only turned into a block when not inside a skipped
				// mc:Choice branch (or similar) — otherwise a table
				// present in both a Choice and its Fallback, or inside a
				// deleted region, would be counted twice or kept at all.
				var raw tblXML
				if err := dec.DecodeElement(&raw, &t); err != nil {
					return parsedDocument{}, ErrUnreadable
				}
				if skipDepth == 0 {
					tbl, tblNotes, err := buildTableBlock(raw, ctx)
					if err != nil {
						return parsedDocument{}, err
					}
					out.blocks = append(out.blocks, block{kind: blockTable, table: tbl})
					out.notes = append(out.notes, tblNotes...)
				}
			case "txbxContent":
				// Same reasoning as "tbl" above: a text box can appear in
				// both an mc:Choice branch and its Fallback (e.g. a
				// modern DrawingML text box alongside its VML fallback),
				// and only the one actually walked (skipDepth == 0)
				// should contribute paragraphs.
				var raw struct {
					InnerXML string `xml:",innerxml"`
				}
				if err := dec.DecodeElement(&raw, &t); err != nil {
					return parsedDocument{}, ErrUnreadable
				}
				if skipDepth == 0 {
					tbDoc, err := parseBlocks(wrapInnerXML(raw.InnerXML), ctx)
					if err != nil {
						return parsedDocument{}, err
					}
					pendingTextBoxParagraphs = append(pendingTextBoxParagraphs, flattenBlocksToParagraphs(tbDoc.blocks)...)
					out.notes = append(out.notes, tbDoc.notes...)
				}
			case "drawing":
				var raw drawingXML
				if err := dec.DecodeElement(&raw, &t); err != nil {
					return parsedDocument{}, ErrUnreadable
				}
				if skipDepth == 0 && inRun {
					flushRun()
					appendMedia(ctx.resolveDrawingSegment(raw))
				}
			case "pict":
				var raw struct {
					InnerXML string `xml:",innerxml"`
				}
				if err := dec.DecodeElement(&raw, &t); err != nil {
					return parsedDocument{}, ErrUnreadable
				}
				if skipDepth == 0 && inRun {
					vml := scanVMLPict(wrapInnerXML(raw.InnerXML))
					switch {
					case vml.txbxInner != "":
						tbDoc, err := parseBlocks(wrapInnerXML(vml.txbxInner), ctx)
						if err != nil {
							return parsedDocument{}, err
						}
						pendingTextBoxParagraphs = append(pendingTextBoxParagraphs, flattenBlocksToParagraphs(tbDoc.blocks)...)
						out.notes = append(out.notes, tbDoc.notes...)
					case vml.imageRelID != "":
						flushRun()
						appendMedia(ctx.resolvePictureSegment(vml.imageRelID, ""))
					default:
						// A drawn shape with no text and no picture (SPEC
						// B4: "drawn shapes with no text").
						flushRun()
						appendMedia(segment{kind: segMedia, missingKind: "shape"})
					}
				}
			case "object":
				// An OLE object (SPEC B4: "embedded objects with no
				// usable image"): typically wraps a VML v:imagedata
				// holding a cached preview, which is used as a real
				// picture when present.
				var raw struct {
					InnerXML string `xml:",innerxml"`
				}
				if err := dec.DecodeElement(&raw, &t); err != nil {
					return parsedDocument{}, ErrUnreadable
				}
				if skipDepth == 0 && inRun {
					flushRun()
					vml := scanVMLPict(wrapInnerXML(raw.InnerXML))
					if vml.imageRelID != "" {
						appendMedia(ctx.resolvePictureSegment(vml.imageRelID, ""))
					} else {
						appendMedia(segment{kind: segMedia, missingKind: "object"})
					}
				}
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
			case "oMath":
				// An equation (SPEC B4). Consumed whole via DecodeElement
				// (into a struct with nothing to capture) rather than
				// left to the normal run walk: m:r/m:t share local names
				// with w:r/w:t, so without this its numerator/denominator
				// text would otherwise leak into the visible text.
				var raw struct{}
				if err := dec.DecodeElement(&raw, &t); err != nil {
					return parsedDocument{}, ErrUnreadable
				}
				if skipDepth == 0 && inParagraph {
					flushRun()
					appendMedia(segment{kind: segMedia, missingKind: "equation"})
				}
			case "commentReference", "footnoteReference", "endnoteReference":
				out.notes = append(out.notes, noteFor(t.Name.Local))
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if inParagraph {
					flushRun()
					out.blocks = append(out.blocks, block{kind: blockParagraph, paragraph: cur})
					for _, tp := range pendingTextBoxParagraphs {
						out.blocks = append(out.blocks, block{kind: blockParagraph, paragraph: tp})
					}
					pendingTextBoxParagraphs = nil
				}
				inParagraph = false
			case "r":
				flushRun()
				inRun = false
			case "rPr":
				inRunProps = false
			case "del", "delText", "moveFrom", "instrText", "Choice":
				if skipDepth > 0 {
					skipDepth--
				}
				if t.Name.Local == "instrText" {
					inInstrText = false
				}
			case "hyperlink":
				if len(hrefStack) > 0 {
					hrefStack = hrefStack[:len(hrefStack)-1]
				}
			case "fldSimple":
				if fldSimplePushedHref {
					hrefStack = hrefStack[:len(hrefStack)-1]
					fldSimplePushedHref = false
				}
			}

		case xml.CharData:
			if inRun && skipDepth == 0 {
				runText.Write(t)
			}
			if inInstrText && fieldState == fieldCollectingInstr {
				fieldInstrBuf.Write(t)
			}
		}
	}

	return out, nil
}

// parseHyperlinkFieldInstr extracts a URL from a HYPERLINK field's
// instruction text (SPEC B4: "HYPERLINK simple or complex fields"), e.g.
// ` HYPERLINK "https://example.com" ` or ` HYPERLINK \l "Bookmark" `. It
// returns ok=false for anything that isn't a recognised external link —
// including an internal-only \l bookmark jump, which SPEC B4 says keeps
// its text and drops the link — or a field that isn't HYPERLINK at all.
func parseHyperlinkFieldInstr(instr string) (url string, ok bool) {
	trimmed := strings.TrimSpace(instr)
	if !strings.HasPrefix(strings.ToUpper(trimmed), "HYPERLINK") {
		return "", false
	}
	rest := trimmed[len("HYPERLINK"):]
	start := strings.IndexByte(rest, '"')
	if start < 0 {
		return "", false
	}
	rest = rest[start+1:]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return "", false
	}
	candidate := rest[:end]
	if isExternalLinkScheme(candidate) {
		return candidate, true
	}
	return "", false
}

func isExternalLinkScheme(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "mailto:")
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
