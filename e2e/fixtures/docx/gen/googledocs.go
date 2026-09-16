package main

// writeGoogleDocsDocx builds a smaller document in the shape Google Docs'
// "Download as .docx" produces: plain English style IDs (no localisation)
// and no Word-specific extras (no header/footer, comments, footnotes,
// chart, SmartArt, or tracked changes) — just title, headings, a list, a
// table and an image, enough to prove the converter isn't accidentally
// tied to Word's own conventions.
func writeGoogleDocsDocx(path string) error {
	png := readFixtureImage("tiny.png")

	body := "" +
		p("Title", run("Office Wi-Fi Guide")) +
		p("Heading1", run("Connecting a laptop")) +
		p("", run("Use the network named ")+boldRun("CompHQ-Staff")+run(" and ask reception for the password.")) +
		p("Heading2", run("Guest access")) +
		pList("ListParagraph", 1, 0, run("Guests use the CompHQ-Guest network")) +
		pList("ListParagraph", 1, 1, run("The password changes weekly")) +
		pList("ListParagraph", 1, 0, run("Ask reception for today's password")) +
		p("", hyperlinkRun("rIdHyperlink1", "network policy")) +
		googleTable() +
		p("", run("Router location:")) +
		image("rIdImage1", "Router in the server cupboard", 1828800, 1371600) +
		p("", run("End of document."))

	docXML := documentXML(body, false)

	docRels := documentRelsXML([]relEntry{
		rel("rId1", "styles", "styles.xml"),
		rel("rId2", "numbering", "numbering.xml"),
		rel("rIdImage1", "image", "media/image1.png"),
		externalRel("rIdHyperlink1", "hyperlink", "https://intranet.example/wifi-policy"),
	})

	parts := []part{
		{"[Content_Types].xml", []byte(contentTypesXML)},
		{"_rels/.rels", []byte(rootRelsXML)},
		{"docProps/core.xml", []byte(coreXML)},
		{"docProps/app.xml", []byte(appXML)},
		{"word/document.xml", []byte(docXML)},
		{"word/_rels/document.xml.rels", []byte(docRels)},
		{"word/styles.xml", []byte(googleDocsStylesXML)},
		{"word/numbering.xml", []byte(numberingXML)},
		{"word/media/image1.png", png},
	}

	return writeDocxFile(path, parts)
}

func googleTable() string {
	return `<w:tbl>` +
		`<w:tblPr><w:tblW w:w="0" w:type="auto"/></w:tblPr>` +
		`<w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>` +
		`<w:tr><w:trPr><w:tblHeader/></w:trPr>` +
		`<w:tc>` + p("", boldRun("Network")) + `</w:tc>` +
		`<w:tc>` + p("", boldRun("Access")) + `</w:tc>` +
		`</w:tr>` +
		`<w:tr>` +
		`<w:tc>` + p("", run("CompHQ-Staff")) + `</w:tc>` +
		`<w:tc>` + p("", run("Staff only")) + `</w:tc>` +
		`</w:tr>` +
		`</w:tbl>`
}
