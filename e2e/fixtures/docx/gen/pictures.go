package main

import (
	"bytes"
	stdimage "image"
	"image/color"
	"image/png"
	"path/filepath"
	"strings"
)

// Fixtures for gates 7.60-7.68 (SPEC B13.9): pictures, in the shapes that went
// wrong at v1.3.0. Every picture is a different, tiny PNG, so a test can tell
// each one from the others by its colour. The PNGs are written uncompressed so
// the bytes never depend on the Go version's compressor.

// pictureColour is the colour of the n-th picture (1-based): distinct for
// n up to a few hundred.
func pictureColour(n int) color.RGBA {
	return color.RGBA{R: uint8(n * 9 % 256), G: uint8(255 - n*7%256), B: uint8(n * 31 % 256), A: 255}
}

// distinctPNG is a 4x4 picture of one colour.
func distinctPNG(n int) []byte {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, 4, 4))
	c := pictureColour(n)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.NoCompression}
	if err := enc.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func baseParts(docXML, docRels string) []part {
	return []part{
		{"[Content_Types].xml", []byte(contentTypesXML)},
		{"_rels/.rels", []byte(rootRelsXML)},
		{"docProps/core.xml", []byte(coreXML)},
		{"docProps/app.xml", []byte(appXML)},
		{"word/document.xml", []byte(docXML)},
		{"word/_rels/document.xml.rels", []byte(docRels)},
		{"word/styles.xml", []byte(wordStylesXML)},
		{"word/numbering.xml", []byte(numberingXML)},
	}
}

// writePictureFixtures writes the picture fixtures into dir.
func writePictureFixtures(dir string) error {
	if err := write25Pictures(filepath.Join(dir, "pictures-25.docx")); err != nil {
		return err
	}
	if err := writeOddPictureNames(filepath.Join(dir, "pictures-oddnames.docx")); err != nil {
		return err
	}
	if err := writePictureInTitle(filepath.Join(dir, "picture-in-title.docx")); err != nil {
		return err
	}
	return writeShapesFixture(filepath.Join(dir, "pictures-shapes.docx"))
}

// pictures-25.docx: 25 different pictures, one after another. At v1.3.0 the
// 10th picture's address was corrupted by the 1st's (gate 7.60).
func write25Pictures(path string) error {
	entries := []relEntry{rel("rId1", "styles", "styles.xml"), rel("rId2", "numbering", "numbering.xml")}
	var media []part
	var body strings.Builder
	body.WriteString(p("Title", run("Twenty-five pictures")))
	for i := 1; i <= 25; i++ {
		id := "rIdPic" + itoa(i)
		name := "media/pic" + itoa(i) + ".png"
		entries = append(entries, rel(id, "image", name))
		media = append(media, part{"word/" + name, distinctPNG(i)})
		body.WriteString(p("", run("Picture "+itoa(i))))
		body.WriteString(p("", image(id, "Picture "+itoa(i), 500000, 500000)))
	}
	parts := append(baseParts(documentXML(body.String(), false), documentRelsXML(entries)), media...)
	return writeDocxFile(path, parts)
}

// pictures-oddnames.docx: three pictures whose targets are written the way
// other programs write them: an absolute path from the package root, a name
// with a space in it (percent-encoded in the target), and a different case
// from the part's real name (gate 7.62).
func writeOddPictureNames(path string) error {
	entries := []relEntry{
		rel("rId1", "styles", "styles.xml"),
		rel("rId2", "numbering", "numbering.xml"),
		rel("rIdAbs", "image", "/word/media/abs.png"),
		rel("rIdSpace", "image", "media/with%20space.png"),
		rel("rIdCase", "image", "media/lower.png"),
	}
	media := []part{
		{"word/media/abs.png", distinctPNG(1)},
		{"word/media/with space.png", distinctPNG(2)},
		{"word/media/LOWER.PNG", distinctPNG(3)},
	}
	var body strings.Builder
	body.WriteString(p("Title", run("Pictures with unusual names")))
	body.WriteString(p("", run("Absolute path")))
	body.WriteString(p("", image("rIdAbs", "Absolute", 500000, 500000)))
	body.WriteString(p("", run("A space in the name")))
	body.WriteString(p("", image("rIdSpace", "Space", 500000, 500000)))
	body.WriteString(p("", run("Upper-case part name")))
	body.WriteString(p("", image("rIdCase", "Case", 500000, 500000)))
	parts := append(baseParts(documentXML(body.String(), false), documentRelsXML(entries)), media...)
	return writeDocxFile(path, parts)
}

// picture-in-title.docx: a picture in the same paragraph as the Title (gate
// 7.63), and one in the body.
func writePictureInTitle(path string) error {
	entries := []relEntry{
		rel("rId1", "styles", "styles.xml"),
		rel("rId2", "numbering", "numbering.xml"),
		rel("rIdLogo", "image", "media/logo.png"),
		rel("rIdBody", "image", "media/body.png"),
	}
	media := []part{
		{"word/media/logo.png", distinctPNG(1)},
		{"word/media/body.png", distinctPNG(2)},
	}
	var body strings.Builder
	body.WriteString(p("Title", run("Quarterly report")+image("rIdLogo", "Logo", 500000, 500000)))
	body.WriteString(p("", run("Some words under the title.")))
	body.WriteString(p("", image("rIdBody", "Chart picture", 500000, 500000)))
	parts := append(baseParts(documentXML(body.String(), false), documentRelsXML(entries)), media...)
	return writeDocxFile(path, parts)
}
