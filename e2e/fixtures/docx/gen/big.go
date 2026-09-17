package main

import "strings"

// writeBig20PagesDocx builds a long document (roughly 20 pages once
// rendered, 10 pictures) for the P1-52 timed-import test. Content is
// generated programmatically rather than hand-written paragraph by
// paragraph.
func writeBig20PagesDocx(path string) error {
	png := readFixtureImage("tiny.png")
	jpg := readFixtureImage("tiny.jpg")

	const sections = 20
	const paragraphsPerSection = 6

	docRelEntries := []relEntry{
		rel("rId1", "styles", "styles.xml"),
		rel("rId2", "numbering", "numbering.xml"),
	}
	var mediaParts []part

	var body strings.Builder
	body.WriteString(p("Title", run("Twenty Section Reference Document")))

	pictureCount := 0
	for i := 1; i <= sections; i++ {
		body.WriteString(p("berschrift1", run("Section "+itoa(i))))
		for j := 1; j <= paragraphsPerSection; j++ {
			body.WriteString(p("", run(strings.Repeat(
				"This is reference text for section "+itoa(i)+", paragraph "+itoa(j)+". ",
				4,
			))))
		}
		if i%2 == 0 && pictureCount < 10 {
			pictureCount++
			relID := "rIdImage" + itoa(pictureCount)
			var mediaName string
			var mediaBytes []byte
			if pictureCount%2 == 0 {
				mediaName = "media/image" + itoa(pictureCount) + ".jpeg"
				mediaBytes = jpg
			} else {
				mediaName = "media/image" + itoa(pictureCount) + ".png"
				mediaBytes = png
			}
			docRelEntries = append(docRelEntries, rel(relID, "image", mediaName))
			mediaParts = append(mediaParts, part{"word/" + mediaName, mediaBytes})
			body.WriteString(p("", run("Figure "+itoa(pictureCount)+":")))
			body.WriteString(p("", image(relID, "Figure "+itoa(pictureCount), 1000000, 750000)))
		}
	}

	docXML := documentXML(body.String(), false)
	docRels := documentRelsXML(docRelEntries)

	parts := []part{
		{"[Content_Types].xml", []byte(contentTypesXML)},
		{"_rels/.rels", []byte(rootRelsXML)},
		{"docProps/core.xml", []byte(coreXML)},
		{"docProps/app.xml", []byte(appXML)},
		{"word/document.xml", []byte(docXML)},
		{"word/_rels/document.xml.rels", []byte(docRels)},
		{"word/styles.xml", []byte(wordStylesXML)},
		{"word/numbering.xml", []byte(numberingXML)},
	}
	parts = append(parts, mediaParts...)

	return writeDocxFile(path, parts)
}
