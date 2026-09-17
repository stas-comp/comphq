// Package docx converts a .docx file to sanitiser-ready HTML. Convert is a
// pure function: no DB, network or disk access (SPEC B2). This is the
// P1-11 spike: title and heading extraction, and the limits that keep a
// hostile or damaged file from ever panicking or exhausting memory.
// Later tasks (P1-29–P1-33) add the rest of SPEC B4's mapping rules
// (marks, lists, links, tables, images, text boxes, tracked changes).
package docx

import (
	"archive/zip"
	"errors"
	"io"
)

// Limits from SPEC B4 "Import from Word".
const (
	MaxUploadSize       = 50 * 1024 * 1024 // 50 MB
	MaxZipEntries       = 5000
	MaxUncompressedSize = 300 * 1024 * 1024 // 300 MB, enforced while reading
	MaxXMLDepth         = 256
)

// ErrUnreadable covers anything that isn't a readable WordprocessingML
// .docx: old .doc (CFB) files, password-protected documents (also CFB),
// PDFs, damaged/truncated zips, and zips missing the parts a .docx needs.
var ErrUnreadable = errors.New("this file can't be read as a Word document")

// ErrTooLarge covers files, entries or structures beyond the limits above,
// including a zip bomb (an entry whose real uncompressed content blows
// past the limit regardless of what its header claims).
var ErrTooLarge = errors.New("this Word document is too large")

// Result is what the KB editor needs to insert the converted document.
// Notes lists one entry per thing left out of the conversion (SPEC B4:
// comments, footnotes, endnotes, headers and footers today; P1-32 adds
// unsupported items like charts and pictures in other formats) — the
// editor turns len(Notes) and its entries into the 1.49 summary message.
type Result struct {
	Title string
	HTML  string
	Notes []string
}

// Convert reads a .docx from r (size bytes long) and returns its title and
// body HTML. filename is used only as a title fallback.
func Convert(r io.ReaderAt, size int64, filename string) (Result, error) {
	if size <= 0 {
		return Result{}, ErrUnreadable
	}
	if size > MaxUploadSize {
		return Result{}, ErrTooLarge
	}

	zr, err := zip.NewReader(r, size)
	if err != nil {
		return Result{}, ErrUnreadable
	}
	if len(zr.File) > MaxZipEntries {
		return Result{}, ErrTooLarge
	}

	pkg := &pkgReader{zr: zr}

	contentTypes, err := pkg.readContentTypes()
	if err != nil {
		return Result{}, err
	}
	if !contentTypes.declaresWordDocument() {
		return Result{}, ErrUnreadable
	}

	mainPart, err := pkg.mainDocumentPart()
	if err != nil {
		return Result{}, err
	}

	docXML, err := pkg.readPart(mainPart)
	if err != nil {
		return Result{}, err
	}

	stylesXML, err := pkg.readPartIfExists(partRelativeTo(mainPart, "styles.xml"))
	if err != nil {
		return Result{}, err
	}
	sheet, err := parseStyles(stylesXML)
	if err != nil {
		return Result{}, err
	}

	numberingXML, err := pkg.readPartIfExists(partRelativeTo(mainPart, "numbering.xml"))
	if err != nil {
		return Result{}, err
	}
	numSheet, err := parseNumbering(numberingXML)
	if err != nil {
		return Result{}, err
	}

	hyperlinkRels, err := pkg.hyperlinkRelationships(mainPart)
	if err != nil {
		return Result{}, err
	}

	parsed, err := parseDocument(docXML, sheet, hyperlinkRels)
	if err != nil {
		return Result{}, err
	}

	hfNotes, err := pkg.headerFooterNotes(mainPart)
	if err != nil {
		return Result{}, err
	}
	notes := append(hfNotes, parsed.notes...)

	return buildResult(parsed.paragraphs, notes, sheet, numSheet, filename), nil
}

// pkgReader reads parts of the zip package, enforcing the uncompressed-size
// limit on every read regardless of what the zip header claims.
type pkgReader struct {
	zr    *zip.Reader
	total int64
}

func (p *pkgReader) findFile(name string) *zip.File {
	for _, f := range p.zr.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func (p *pkgReader) readPart(name string) ([]byte, error) {
	f := p.findFile(name)
	if f == nil {
		return nil, ErrUnreadable
	}

	// The zip header's uncompressed-size field isn't trusted (a zip bomb
	// can lie about it, and Go's own zip reader already refuses a header
	// that doesn't match the data); the limit is enforced on bytes
	// actually decompressed.
	remaining := MaxUncompressedSize - p.total
	if remaining <= 0 {
		return nil, ErrTooLarge
	}

	// First pass: find the true decompressed size by discarding it as we
	// go, so rejecting an oversized part costs only a small, constant
	// amount of memory (io.Copy's own buffer) no matter how large the
	// part claims, or turns out, to be.
	probe, err := f.Open()
	if err != nil {
		return nil, ErrUnreadable
	}
	n, copyErr := io.Copy(io.Discard, io.LimitReader(probe, remaining+1))
	probe.Close()
	if copyErr != nil {
		return nil, ErrUnreadable
	}
	if n > remaining {
		return nil, ErrTooLarge
	}

	// Second pass: it's within budget, so actually read it.
	rc, err := f.Open()
	if err != nil {
		return nil, ErrUnreadable
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, ErrUnreadable
	}
	p.total += int64(len(data))
	return data, nil
}

func (p *pkgReader) readPartIfExists(name string) ([]byte, error) {
	if p.findFile(name) == nil {
		return nil, nil
	}
	return p.readPart(name)
}

func (p *pkgReader) readContentTypes() (contentTypes, error) {
	data, err := p.readPart("[Content_Types].xml")
	if err != nil {
		return contentTypes{}, err
	}
	return parseContentTypes(data)
}

// mainDocumentPart finds the main document part via _rels/.rels (SPEC B4:
// "Find the main part through _rels/.rels, not a fixed path"), rather than
// assuming word/document.xml.
func (p *pkgReader) mainDocumentPart() (string, error) {
	data, err := p.readPart("_rels/.rels")
	if err != nil {
		return "", err
	}
	rels, err := parseRelationships(data)
	if err != nil {
		return "", err
	}
	for _, rel := range rels {
		if isOfficeDocumentRelType(rel.Type) {
			return normalizePartPath(rel.Target), nil
		}
	}
	return "", ErrUnreadable
}

// headerFooterNotes returns one note per header or footer relationship
// found for mainPart (SPEC B4: "left out, and counted in notes"). Headers
// and footers are separate parts referenced by relationship, unlike
// comments/footnotes/endnotes, which parseDocument already counts from
// their inline references in document.xml.
func (p *pkgReader) headerFooterNotes(mainPart string) ([]string, error) {
	data, err := p.readPartIfExists(relsPathFor(mainPart))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	rels, err := parseRelationships(data)
	if err != nil {
		return nil, err
	}
	var notes []string
	for _, rel := range rels {
		switch {
		case hasSuffixFold(rel.Type, "/header"):
			notes = append(notes, "the header")
		case hasSuffixFold(rel.Type, "/footer"):
			notes = append(notes, "the footer")
		}
	}
	return notes, nil
}

// hyperlinkRelationships returns mainPart's hyperlink relationships as a
// map from relationship id to target URL, for w:hyperlink r:id lookups
// (SPEC B4: "w:hyperlink with an external relationship").
func (p *pkgReader) hyperlinkRelationships(mainPart string) (map[string]string, error) {
	data, err := p.readPartIfExists(relsPathFor(mainPart))
	if err != nil {
		return nil, err
	}
	rels := map[string]string{}
	if data == nil {
		return rels, nil
	}
	parsed, err := parseRelationships(data)
	if err != nil {
		return nil, err
	}
	for _, r := range parsed {
		if hasSuffixFold(r.Type, "/hyperlink") {
			rels[r.ID] = r.Target
		}
	}
	return rels, nil
}

// relsPathFor returns the relationships part for part, e.g.
// "word/document.xml" -> "word/_rels/document.xml.rels".
func relsPathFor(part string) string {
	dir, base := "", part
	for i := len(part) - 1; i >= 0; i-- {
		if part[i] == '/' {
			dir, base = part[:i+1], part[i+1:]
			break
		}
	}
	return dir + "_rels/" + base + ".rels"
}

func isOfficeDocumentRelType(relType string) bool {
	return hasSuffixFold(relType, "/officeDocument")
}

func hasSuffixFold(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	tail := s[len(s)-len(suffix):]
	return equalFold(tail, suffix)
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// normalizePartPath turns a relationship target like "word/document.xml"
// or "/word/document.xml" into the zip entry name "word/document.xml".
func normalizePartPath(target string) string {
	for len(target) > 0 && target[0] == '/' {
		target = target[1:]
	}
	return target
}

// partRelativeTo resolves name relative to the directory containing part,
// e.g. partRelativeTo("word/document.xml", "styles.xml") == "word/styles.xml".
func partRelativeTo(part, name string) string {
	dir := ""
	for i := len(part) - 1; i >= 0; i-- {
		if part[i] == '/' {
			dir = part[:i+1]
			break
		}
	}
	return dir + name
}
