package main

import "strings"

// pictures-shapes.docx (gates 7.61, 7.65, 7.66; SPEC B13.9 items 2, 3, 4, 9,
// 10): every way Word puts a picture somewhere other than a plain inline one.
// Each is introduced by a caption so a failure says which one went wrong, and
// every picture is a different colour (see pictures.go) so the order can be
// asserted. In document order the pictures are:
//
//	 1       an anchored (floating) picture
//	 2-4     a DrawingML group of three pictures
//	 5-6     a canvas holding two pictures
//	 7-9     a VML group of three pictures
//	 10      a VML shape filled with a picture (v:fill)
//	 11      a picture in an mc:Choice with no Fallback
//	 12      the same picture in both a Choice and its VML Fallback (counts once)
//	 13      a Choice with a picture beside a Fallback with none
//	 (EMF)   an old Windows drawing, which can only be marked
//
// and the page header holds a logo, which is left out and counted.

const (
	nsWPG = `xmlns:wpg="http://schemas.microsoft.com/office/word/2010/wordprocessingGroup"`
	nsWPC = `xmlns:wpc="http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas"`
)

func picPic(relID, alt string) string {
	return `<pic:pic><pic:nvPicPr><pic:cNvPr id="1" name="Picture" descr="` + escapeXML(alt) + `"/><pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip r:embed="` + relID + `"/></pic:blipFill><pic:spPr/></pic:pic>`
}

func graphicInline(uri, inner, alt string) string {
	return `<wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="900000" cy="900000"/>` +
		`<wp:docPr id="7" name="Drawing" descr="` + escapeXML(alt) + `"/>` +
		`<a:graphic><a:graphicData uri="` + uri + `">` + inner + `</a:graphicData></a:graphic></wp:inline>`
}

func anchorImage(relID, alt string) string {
	return `<w:r><w:drawing><wp:anchor distT="0" distB="0" distL="114300" distR="114300" simplePos="0" relativeHeight="1" behindDoc="0" locked="0" layoutInCell="1" allowOverlap="1">` +
		`<wp:simplePos x="0" y="0"/><wp:positionH relativeFrom="column"><wp:posOffset>0</wp:posOffset></wp:positionH>` +
		`<wp:positionV relativeFrom="paragraph"><wp:posOffset>0</wp:posOffset></wp:positionV>` +
		`<wp:extent cx="900000" cy="900000"/><wp:wrapSquare wrapText="bothSides"/>` +
		`<wp:docPr id="3" name="Floating" descr="` + escapeXML(alt) + `"/>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` + picPic(relID, alt) + `</a:graphicData></a:graphic></wp:anchor></w:drawing></w:r>`
}

func drawingGroup(relIDs []string) string {
	var inner strings.Builder
	inner.WriteString(`<wpg:wgp ` + nsWPG + `><wpg:cNvGrpSpPr/><wpg:grpSpPr/>`)
	for i, id := range relIDs {
		inner.WriteString(picPic(id, "Grouped "+itoa(i+1)))
	}
	inner.WriteString(`</wpg:wgp>`)
	return `<w:r><w:drawing>` + graphicInline("http://schemas.microsoft.com/office/word/2010/wordprocessingGroup", inner.String(), "A group of pictures") + `</w:drawing></w:r>`
}

func drawingCanvas(relIDs []string) string {
	var inner strings.Builder
	inner.WriteString(`<wpc:wpc ` + nsWPC + `><wpc:bg/><wpc:whole/>`)
	for i, id := range relIDs {
		inner.WriteString(picPic(id, "On the canvas "+itoa(i+1)))
	}
	inner.WriteString(`</wpc:wpc>`)
	return `<w:r><w:drawing>` + graphicInline("http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas", inner.String(), "A drawing canvas") + `</w:drawing></w:r>`
}

func vmlGroup(relIDs []string) string {
	var inner strings.Builder
	inner.WriteString(`<w:r><w:pict><v:group id="g1" style="width:150pt;height:50pt" coordsize="3000,1000">`)
	for i, id := range relIDs {
		inner.WriteString(`<v:shape id="s` + itoa(i+1) + `" type="#_x0000_t75" style="width:50pt;height:50pt"><v:imagedata r:id="` + id + `"/></v:shape>`)
	}
	inner.WriteString(`</v:group></w:pict></w:r>`)
	return inner.String()
}

func vmlFill(relID string) string {
	return `<w:r><w:pict><v:rect id="r1" style="width:80pt;height:60pt" stroked="f"><v:fill r:id="` + relID + `" type="frame"/></v:rect></w:pict></w:r>`
}

func choiceDrawing(drawing, fallback string) string {
	out := `<w:r><mc:AlternateContent><mc:Choice Requires="wps">` + drawing + `</mc:Choice>`
	if fallback != "" {
		out += `<mc:Fallback>` + fallback + `</mc:Fallback>`
	}
	return out + `</mc:AlternateContent></w:r>`
}

// inlineDrawing is image() without its enclosing run, for use inside a Choice.
func inlineDrawing(relID, alt string) string {
	return strings.TrimSuffix(strings.TrimPrefix(image(relID, alt, 900000, 900000), "<w:r>"), "</w:r>")
}

const headerWithLogoXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">
<w:p>LOGO</w:p>
</w:hdr>`

func writeShapesFixture(path string) error {
	entries := []relEntry{
		rel("rId1", "styles", "styles.xml"),
		rel("rId2", "numbering", "numbering.xml"),
		rel("rIdHeader1", "header", "header1.xml"),
		rel("rIdFooter1", "footer", "footer1.xml"),
	}
	var media []part
	next := 0
	add := func() string { // a new, distinct picture; returns its relationship id
		next++
		id := "rIdP" + itoa(next)
		name := "media/p" + itoa(next) + ".png"
		entries = append(entries, rel(id, "image", name))
		media = append(media, part{"word/" + name, distinctPNG(next)})
		return id
	}
	adds := func(n int) []string {
		var ids []string
		for i := 0; i < n; i++ {
			ids = append(ids, add())
		}
		return ids
	}

	var body strings.Builder
	body.WriteString(p("Title", run("Pictures in every kind of shape")))
	section := func(caption, runs string) {
		body.WriteString(p("", run(caption)))
		body.WriteString(p("", runs))
	}
	section("Floating picture", anchorImage(add(), "Floating"))
	section("A group of three", drawingGroup(adds(3)))
	section("A canvas of two", drawingCanvas(adds(2)))
	section("A VML group of three", vmlGroup(adds(3)))
	section("A picture-filled shape", vmlFill(add()))
	section("A Choice with no Fallback", choiceDrawing(inlineDrawing(add(), "Choice only"), ""))

	// The same picture, once in a Choice and again in its Fallback: one picture.
	same := add()
	section("The same picture in a Choice and its Fallback",
		choiceDrawing(inlineDrawing(same, "Both"), `<w:pict><v:shape id="fb" type="#_x0000_t75" style="width:50pt;height:50pt"><v:imagedata r:id="`+same+`"/></v:shape></w:pict>`))

	// A Choice with a picture, beside a Fallback that shows none.
	section("A Choice with a picture beside an empty Fallback",
		choiceDrawing(inlineDrawing(add(), "Better in the Choice"), `<w:t>Fallback text only</w:t>`))

	// An EMF: can only be marked.
	entries = append(entries, rel("rIdEmf", "image", "media/drawing.emf"))
	media = append(media, part{"word/media/drawing.emf", emfBytes})
	section("An old Windows drawing", image("rIdEmf", "A Visio drawing", 900000, 900000))

	headerRels := documentRelsXML([]relEntry{rel("rIdLogo", "image", "media/logo.png")})
	headerXML := strings.Replace(headerWithLogoXML, "LOGO", `<w:r><w:t>Letterhead</w:t></w:r>`+image("rIdLogo", "Logo", 500000, 500000), 1)

	parts := baseParts(documentXML(body.String(), true), documentRelsXML(entries))
	parts = append(parts,
		part{"word/header1.xml", []byte(headerXML)},
		part{"word/_rels/header1.xml.rels", []byte(headerRels)},
		part{"word/footer1.xml", []byte(footerXML)},
		part{"word/media/logo.png", distinctPNG(99)},
	)
	parts = append(parts, media...)
	return writeDocxFile(path, parts)
}
