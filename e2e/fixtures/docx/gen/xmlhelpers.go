package main

import "strings"

func escapeXML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

// p wraps runsXML in a paragraph using styleID (empty for the default style).
func p(styleID, runsXML string) string {
	if styleID == "" {
		return "<w:p>" + runsXML + "</w:p>"
	}
	return "<w:p><w:pPr><w:pStyle w:val=\"" + styleID + "\"/></w:pPr>" + runsXML + "</w:p>"
}

// pList wraps runsXML in a paragraph with both a style and list numbering
// (numId/ilvl), for nested bullet/number list items.
func pList(styleID string, numID, ilvl int, runsXML string) string {
	return "<w:p><w:pPr><w:pStyle w:val=\"" + styleID + "\"/>" +
		"<w:numPr><w:ilvl w:val=\"" + itoa(ilvl) + "\"/><w:numId w:val=\"" + itoa(numID) + "\"/></w:numPr>" +
		"</w:pPr>" + runsXML + "</w:p>"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

// run is a plain text run.
func run(text string) string {
	return "<w:r><w:t xml:space=\"preserve\">" + escapeXML(text) + "</w:t></w:r>"
}

func boldRun(text string) string {
	return "<w:r><w:rPr><w:b/></w:rPr><w:t xml:space=\"preserve\">" + escapeXML(text) + "</w:t></w:r>"
}

func italicRun(text string) string {
	return "<w:r><w:rPr><w:i/></w:rPr><w:t xml:space=\"preserve\">" + escapeXML(text) + "</w:t></w:r>"
}

// boldOffRun demonstrates SPEC B4's "honouring w:val=0/false" rule: the
// <w:b> element is present but explicitly turned off, unlike a run that
// omits <w:b> entirely.
func boldOffRun(text string) string {
	return "<w:r><w:rPr><w:b w:val=\"0\"/></w:rPr><w:t xml:space=\"preserve\">" + escapeXML(text) + "</w:t></w:r>"
}

// runSplitMidWord splits text into two runs mid-word (a common real-world
// artifact of Word's spell-check/revision bookkeeping), which the
// converter must still read as one word.
func runSplitMidWord(firstHalf, secondHalf string) string {
	return run(firstHalf) + run(secondHalf)
}

func tab() string {
	return "<w:r><w:tab/></w:r>"
}

func lineBreak() string {
	return "<w:r><w:br/></w:r>"
}

func insertedRun(author, text string) string {
	return `<w:ins w:id="900" w:author="` + author + `" w:date="2026-01-01T00:00:00Z">` + run(text) + `</w:ins>`
}

func deletedRun(author, text string) string {
	return `<w:del w:id="901" w:author="` + author + `" w:date="2026-01-01T00:00:00Z"><w:r><w:delText xml:space="preserve">` +
		escapeXML(text) + `</w:delText></w:r></w:del>`
}

func hyperlinkRun(relID, text string) string {
	return `<w:hyperlink r:id="` + relID + `">` +
		`<w:r><w:rPr><w:rStyle w:val="Internetlink"/></w:rPr><w:t xml:space="preserve">` +
		escapeXML(text) + `</w:t></w:r></w:hyperlink>`
}

// hyperlinkField is the legacy "complex field" way of encoding a link,
// still produced by some tools and by copy-paste from other Office apps.
func hyperlinkFieldRun(url, text string) string {
	return `<w:r><w:fldChar w:fldCharType="begin"/></w:r>` +
		`<w:r><w:instrText xml:space="preserve"> HYPERLINK "` + escapeXML(url) + `" </w:instrText></w:r>` +
		`<w:r><w:fldChar w:fldCharType="separate"/></w:r>` +
		run(text) +
		`<w:r><w:fldChar w:fldCharType="end"/></w:r>`
}

func commentedRun(commentID, text string) string {
	return `<w:commentRangeStart w:id="` + commentID + `"/>` + run(text) +
		`<w:commentRangeEnd w:id="` + commentID + `"/>` +
		`<w:r><w:commentReference w:id="` + commentID + `"/></w:r>`
}

func footnoteRun(text string) string {
	return run(text) + `<w:r><w:rPr><w:rStyle w:val="FootnoteReference"/></w:rPr><w:footnoteReference w:id="1"/></w:r>`
}

// image is an inline drawing (wp:inline / a:blip), with alt text from
// wp:docPr's descr attribute (SPEC B4).
func image(relID, altText string, widthEMU, heightEMU int) string {
	return `<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">` +
		`<wp:extent cx="` + itoa(widthEMU) + `" cy="` + itoa(heightEMU) + `"/>` +
		`<wp:docPr id="1" name="Picture" descr="` + escapeXML(altText) + `"/>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic><pic:blipFill><a:blip r:embed="` + relID + `"/></pic:blipFill>` +
		`<pic:spPr><a:xfrm><a:ext cx="` + itoa(widthEMU) + `" cy="` + itoa(heightEMU) + `"/></a:xfrm></pic:spPr>` +
		`</pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`
}

// chartFrame embeds a chart via a graphicFrame referencing a chart part
// (SPEC B4: unsupported items become a placeholder — charts included).
func chartFrame(relID string) string {
	return `<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">` +
		`<wp:extent cx="4572000" cy="2286000"/>` +
		`<wp:docPr id="2" name="Chart"/>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/chart">` +
		`<c:chart xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart" r:id="` + relID + `"/>` +
		`</a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`
}

// smartArtWithFallback represents SmartArt as mc:AlternateContent (SPEC
// B4: "Inside mc:AlternateContent, use the one branch that yields an
// image or text, never both"). The Choice needs a full diagram data
// model to be genuinely valid; the Fallback branch (the one a converter
// should actually use here) is plain text, which is enough to prove the
// "pick exactly one branch" rule without hand-building all four SmartArt
// parts for a fixture the converter treats as unsupported either way.
func smartArtWithFallback() string {
	return `<w:r><mc:AlternateContent xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006">` +
		`<mc:Choice Requires="dgm"><w:drawing><wp:inline><wp:extent cx="1" cy="1"/>` +
		`<wp:docPr id="3" name="SmartArt"/><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/diagram">` +
		`<dgm:relIds xmlns:dgm="http://schemas.openxmlformats.org/drawingml/2006/diagram" r:dm="rIdSmartArtData" r:lo="rIdSmartArtData" r:qs="rIdSmartArtData" r:cs="rIdSmartArtData"/>` +
		`</a:graphicData></a:graphic></wp:inline></w:drawing></mc:Choice>` +
		`<mc:Fallback>` + run("[SmartArt: office chain of command]") + `</mc:Fallback>` +
		`</mc:AlternateContent></w:r>`
}

// drawnShapeNoText is a VML autoshape with no text content (SPEC B4: one
// of the "unsupported items" kinds — "drawn shapes with no text").
func drawnShapeNoText() string {
	return `<w:r><w:pict xmlns:v="urn:schemas-microsoft-com:vml">` +
		`<v:rect id="shape1" style="width:50pt;height:50pt" fillcolor="#D9531E"/>` +
		`</w:pict></w:r>`
}

// textBox is a legacy VML shape carrying its own paragraph content (SPEC
// B4: "Text boxes (w:txbxContent): output their paragraphs after the
// paragraph that anchors them").
func textBox(text string) string {
	return `<w:r><w:pict xmlns:v="urn:schemas-microsoft-com:vml">` +
		`<v:shape id="textbox1" style="width:200pt;height:40pt">` +
		`<v:textbox><w:txbxContent>` + p("", run(text)) + `</w:txbxContent></v:textbox>` +
		`</v:shape></w:pict></w:r>`
}

// equation is an inline OMML formula (SPEC B4: unsupported item "equation").
func equation() string {
	return `<m:oMath xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math">` +
		`<m:f><m:fPr/><m:num><m:r><m:t>1</m:t></m:r></m:num><m:den><m:r><m:t>2</m:t></m:r></m:den></m:f>` +
		`</m:oMath>`
}
