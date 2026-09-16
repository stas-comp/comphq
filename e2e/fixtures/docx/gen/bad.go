package main

import (
	"bytes"
	"os"
	"path/filepath"
)

// writeBadFixtures builds the fixtures SPEC B4's limits and unreadable-
// file handling are tested against (PLAN.md P1-11).
func writeBadFixtures(dir string) error {
	if err := os.WriteFile(filepath.Join(dir, "oldword.doc"), minimalCFB(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "protected.docx"), minimalCFB(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "file.pdf"), minimalPDF(), 0o644); err != nil {
		return err
	}
	if err := writeTruncatedDocx(filepath.Join(dir, "truncated.docx")); err != nil {
		return err
	}
	if err := writeZipBombDocx(filepath.Join(dir, "zipbomb.docx")); err != nil {
		return err
	}
	if err := writeDeepDocx(filepath.Join(dir, "deep.docx")); err != nil {
		return err
	}
	return nil
}

// minimalCFB is just the OLE/Compound File Binary signature plus padding
// to a full 512-byte header sector. It stands in for both a legacy .doc
// and a password-protected .docx (Word writes encrypted files as CFB
// wrapping an encrypted package, per MS-OFFCRYPTO) — either way, it's not
// a zip, so Convert must reject it as ErrUnreadable, not panic.
func minimalCFB() []byte {
	sig := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	b := make([]byte, 512)
	copy(b, sig)
	return b
}

func minimalPDF() []byte {
	return []byte("%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF")
}

// writeTruncatedDocx reuses the real sample.docx bytes cut short, so the
// zip's central directory is missing — a damaged zip, not a clean parse
// failure.
func writeTruncatedDocx(path string) error {
	tmp := filepath.Join(filepath.Dir(filepath.Dir(path)), "sample.docx")
	full, err := os.ReadFile(tmp)
	if err != nil {
		return err
	}
	cut := len(full) * 3 / 5
	return os.WriteFile(path, full[:cut], 0o644)
}

// writeZipBombDocx builds an otherwise-normal package whose word/document.xml
// entry is a real, honestly-labelled zip bomb: 320 MB of a single repeated
// byte compresses to well under a megabyte, but the header (correctly)
// declares the full uncompressed size — comfortably past the converter's
// 300 MB total-uncompressed limit. This is what a real zip bomb looks
// like: no dishonest header is needed (and Go's zip reader actively
// rejects one that doesn't match the data actually read).
func writeZipBombDocx(path string) error {
	payload := bytes.Repeat([]byte{0}, 320*1024*1024)

	parts := []part{
		{"[Content_Types].xml", []byte(contentTypesXML)},
		{"_rels/.rels", []byte(rootRelsXML)},
		{"word/_rels/document.xml.rels", []byte(documentRelsXML([]relEntry{
			rel("rId1", "styles", "styles.xml"),
		}))},
		{"word/styles.xml", []byte(wordStylesXML)},
		{"word/document.xml", payload},
	}
	return writeDocxFile(path, parts)
}

// writeDeepDocx builds a normal package whose word/document.xml has 300
// levels of element nesting — past the converter's 256-deep limit — so
// unmarshalling it can never blow the stack.
func writeDeepDocx(path string) error {
	const depth = 300
	var openTags, closeTags bytes.Buffer
	for i := 0; i < depth; i++ {
		openTags.WriteString("<x>")
		closeTags.WriteString("</x>")
	}
	deepXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		openTags.String() + "deep" + closeTags.String() +
		`</w:body></w:document>`

	docRels := documentRelsXML([]relEntry{rel("rId1", "styles", "styles.xml")})

	parts := []part{
		{"[Content_Types].xml", []byte(contentTypesXML)},
		{"_rels/.rels", []byte(rootRelsXML)},
		{"word/_rels/document.xml.rels", []byte(docRels)},
		{"word/styles.xml", []byte(wordStylesXML)},
		{"word/document.xml", []byte(deepXML)},
	}
	return writeDocxFile(path, parts)
}
