package main

// writeSampleDocx builds a realistic, Word-365-shaped document covering
// every B4 mapping rule listed in PLAN.md P1-11: title, H1–H3, bold/italic
// (including the w:val="0" case and a run split mid-word), nested bullets
// and numbers, a relationship hyperlink and a legacy HYPERLINK field, a
// table with a header row, a column span and a row span, three images
// with alt text, a chart, SmartArt (as mc:AlternateContent), a shape with
// no text, an equation, an EMF picture, a text box, tracked insert and
// delete, a comment, a footnote, and a header and footer.
func writeSampleDocx(path string) error {
	png := readFixtureImage("tiny.png")
	jpg := readFixtureImage("tiny.jpg")

	body := "" +
		p("Title", run("Printer Supplies Handbook")) +
		p("berschrift1", run("Getting started")) +
		p("", run("This guide explains how to keep the office printers running. Ask ")+
			boldRun("Sam")+run(" before ordering more ")+italicRun("toner")+run(".")) +
		p("", run("A run can be split ")+runSplitMidWord("mid", "word")+run(" and still read as one word.")) +
		p("", boldOffRun("This text is explicitly not bold, even though a <w:b> element is present.")) +
		p("berschrift2", run("Ordering toner")) +
		pList("Listenabsatz", 1, 0, run("Check the cupboard first")) +
		pList("Listenabsatz", 1, 1, run("Look on the top shelf")) +
		pList("Listenabsatz", 1, 0, run("Note the model number")) +
		pList("Listenabsatz", 2, 0, run("Email Sam with the model number")) +
		pList("Listenabsatz", 2, 1, run("cc the office manager")) +
		pList("Listenabsatz", 2, 0, run("Wait for confirmation before ordering")) +
		p("", hyperlinkRun("rIdHyperlink1", "the supplier's ordering page")) +
		p("", hyperlinkFieldRun("https://printers.example/manual", "full manual")) +
		p("berschrift3", run("Model reference")) +
		table() +
		p("", commentedRun("0", "the current stock levels")) +
		p("", run("Prices last checked")+footnoteRun(" last quarter")) +
		p("", run("Diagram of the toner compartment:")) +
		image("rIdImage1", "Diagram of the printer's toner compartment", 2286000, 1714500) +
		p("", run("Photo of a spare cartridge:")) +
		image("rIdImage2", "Spare toner cartridge on a shelf", 1828800, 1371600) +
		p("", run("A scanned label (EMF):")) +
		image("rIdImage3", "Scanned supplier label", 1000000, 500000) +
		p("", run("Monthly usage:")) +
		chartFrame("rIdChart") +
		p("", run("Office structure:")) +
		smartArtWithFallback() +
		p("", run("A decorative shape:")) +
		drawnShapeNoText() +
		p("", run("A quick calculation:")) +
		p("", equationParagraph()) +
		p("", textBox("Note: check toner levels weekly.")) +
		p("", run("Tracked changes: ")+insertedRun("Sam", "please order two boxes")+deletedRun("Sam", "order one box")) +
		p("", run("End of document."))

	docXML := documentXML(body, true)

	docRels := documentRelsXML([]relEntry{
		rel("rId1", "styles", "styles.xml"),
		rel("rId2", "numbering", "numbering.xml"),
		rel("rIdHeader1", "header", "header1.xml"),
		rel("rIdFooter1", "footer", "footer1.xml"),
		rel("rIdComments", "comments", "comments.xml"),
		rel("rIdFootnotes", "footnotes", "footnotes.xml"),
		rel("rIdImage1", "image", "media/image1.png"),
		rel("rIdImage2", "image", "media/image2.jpeg"),
		rel("rIdImage3", "image", "media/image3.emf"),
		rel("rIdChart", "chart", "charts/chart1.xml"),
		externalRel("rIdHyperlink1", "hyperlink", "https://printers.example/order"),
	})

	parts := []part{
		{"[Content_Types].xml", []byte(contentTypesXML)},
		{"_rels/.rels", []byte(rootRelsXML)},
		{"docProps/core.xml", []byte(coreXML)},
		{"docProps/app.xml", []byte(appXML)},
		{"word/document.xml", []byte(docXML)},
		{"word/_rels/document.xml.rels", []byte(docRels)},
		{"word/styles.xml", []byte(wordStylesXML)},
		{"word/numbering.xml", []byte(numberingXML)},
		{"word/header1.xml", []byte(headerXML)},
		{"word/footer1.xml", []byte(footerXML)},
		{"word/comments.xml", []byte(commentsXML)},
		{"word/footnotes.xml", []byte(footnotesXML)},
		{"word/charts/chart1.xml", []byte(chart1XML)},
		{"word/media/image1.png", png},
		{"word/media/image2.jpeg", jpg},
		{"word/media/image3.emf", emfBytes},
	}

	return writeDocxFile(path, parts)
}

// equationParagraph wraps an inline OMML formula for its own paragraph.
func equationParagraph() string {
	return equation()
}

func table() string {
	return `<w:tbl>` +
		`<w:tblPr><w:tblW w:w="0" w:type="auto"/></w:tblPr>` +
		`<w:tblGrid><w:gridCol/><w:gridCol/><w:gridCol/></w:tblGrid>` +
		// Header row: "Model" spans two columns (gridSpan), "Toner code" is its own column.
		`<w:tr><w:trPr><w:tblHeader/></w:trPr>` +
		`<w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr>` + p("", boldRun("Model")) + `</w:tc>` +
		`<w:tc>` + p("", boldRun("Toner code")) + `</w:tc>` +
		`</w:tr>` +
		// Data row 1: "LX-4200" starts a vertical merge across two rows.
		`<w:tr>` +
		`<w:tc><w:tcPr><w:gridSpan w:val="2"/><w:vMerge w:val="restart"/></w:tcPr>` + p("", run("LX-4200")) + `</w:tc>` +
		`<w:tc>` + p("", run("TN-820")) + `</w:tc>` +
		`</w:tr>` +
		// Data row 2: the merged cell continues (empty content, per OOXML).
		`<w:tr>` +
		`<w:tc><w:tcPr><w:gridSpan w:val="2"/><w:vMerge/></w:tcPr>` + p("", run("")) + `</w:tc>` +
		`<w:tc>` + p("", run("TN-821")) + `</w:tc>` +
		`</w:tr>` +
		`</w:tbl>`
}
