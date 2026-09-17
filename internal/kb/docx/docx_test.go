package docx

import (
	"archive/zip"
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func fixturesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	// internal/kb/docx/docx_test.go -> repository root -> e2e/fixtures/docx
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(root, "e2e", "fixtures", "docx")
}

func convertFixture(t *testing.T, relPath string) (Result, error) {
	t.Helper()
	path := filepath.Join(fixturesDir(t), relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", relPath, err)
	}
	return Convert(bytes.NewReader(data), int64(len(data)), filepath.Base(relPath))
}

func TestConvertSampleDocxTitleAndHeadings(t *testing.T) {
	result, err := convertFixture(t, "sample.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}

	if result.Title != "Printer Supplies Handbook" {
		t.Errorf("Title = %q, want %q", result.Title, "Printer Supplies Handbook")
	}

	for _, want := range []string{
		"<h2>Getting started</h2>",
		"<h3>Ordering toner</h3>", // heading 2 (SPEC B4: "heading 2–9 → h3")
		"<h3>Model reference</h3>",
	} {
		if !strings.Contains(result.HTML, want) {
			t.Errorf("HTML missing %q; got:\n%s", want, result.HTML)
		}
	}

	// The title paragraph must be removed from the body, not duplicated.
	if strings.Contains(result.HTML, "Printer Supplies Handbook") {
		t.Error("title text should be removed from the body HTML")
	}
}

func TestConvertGoogleDocsDocxTitleAndHeadings(t *testing.T) {
	result, err := convertFixture(t, "googledocs.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}

	if result.Title != "Office Wi-Fi Guide" {
		t.Errorf("Title = %q, want %q", result.Title, "Office Wi-Fi Guide")
	}

	for _, want := range []string{
		"<h2>Connecting a laptop</h2>",
		"<h3>Guest access</h3>",
	} {
		if !strings.Contains(result.HTML, want) {
			t.Errorf("HTML missing %q; got:\n%s", want, result.HTML)
		}
	}
}

func TestConvertBadFixtures(t *testing.T) {
	cases := []struct {
		file    string
		wantErr error
	}{
		{"bad/oldword.doc", ErrUnreadable},
		{"bad/protected.docx", ErrUnreadable},
		{"bad/file.pdf", ErrUnreadable},
		{"bad/truncated.docx", ErrUnreadable},
		{"bad/zipbomb.docx", ErrTooLarge},
		{"bad/deep.docx", ErrTooLarge},
	}

	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			path := filepath.Join(fixturesDir(t), c.file)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			done := make(chan struct{})
			var (
				result Result
				gotErr error
			)
			start := time.Now()
			go func() {
				defer close(done)
				result, gotErr = Convert(bytes.NewReader(data), int64(len(data)), filepath.Base(c.file))
			}()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatalf("Convert(%s) did not return within 2s", c.file)
			}
			elapsed := time.Since(start)

			if gotErr != c.wantErr {
				t.Errorf("Convert(%s) error = %v, want %v", c.file, gotErr, c.wantErr)
			}
			if result.Title != "" || result.HTML != "" {
				t.Errorf("Convert(%s) returned a non-empty Result alongside an error: %+v", c.file, result)
			}
			if elapsed > 2*time.Second {
				t.Errorf("Convert(%s) took %s, want <= 2s", c.file, elapsed)
			}
		})
	}
}

// TestConvertZipBombUsesBoundedMemory proves rejecting a bomb doesn't
// require materialising anywhere near its (claimed or real) size: PLAN.md
// P1-11 asks for "under 64 MB allocation ... use a size-limited reader
// test" rather than testing.AllocsPerRun.
func TestConvertZipBombUsesBoundedMemory(t *testing.T) {
	path := filepath.Join(fixturesDir(t), "bad", "zipbomb.docx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	_, convertErr := Convert(bytes.NewReader(data), int64(len(data)), "zipbomb.docx")

	runtime.ReadMemStats(&after)

	if convertErr != ErrTooLarge {
		t.Fatalf("Convert error = %v, want ErrTooLarge", convertErr)
	}
	const limit = 64 * 1024 * 1024
	if grew := after.TotalAlloc - before.TotalAlloc; grew > limit {
		t.Errorf("Convert allocated %d bytes rejecting a bomb, want <= %d", grew, limit)
	}
}

func TestConvertRejectsOversizeUpload(t *testing.T) {
	data := make([]byte, MaxUploadSize+1)
	_, err := Convert(bytes.NewReader(data), int64(len(data)), "huge.docx")
	if err != ErrTooLarge {
		t.Errorf("Convert of an oversize upload: err = %v, want ErrTooLarge", err)
	}
}

func TestTitleFromFilenameFallback(t *testing.T) {
	// A minimal valid .docx with no Title style and no heading 1: the
	// title must fall back to the filename without its extension (SPEC B4).
	data := buildMinimalDocxNoTitle(t, "Just a plain paragraph, nothing special.")
	result, err := Convert(bytes.NewReader(data), int64(len(data)), "Meeting Notes.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if result.Title != "Meeting Notes" {
		t.Errorf("Title = %q, want %q", result.Title, "Meeting Notes")
	}
}

// buildDocx assembles a minimal but valid .docx package for one narrowly
// scoped unit test: document.xml (with the given body) and, if non-empty,
// styles.xml — no header/footer/comments. Building it in Go rather than
// committing a binary satisfies PLAN.md's "no hand-edited binaries" for
// rules narrow enough not to need the shared, realistic
// e2e/fixtures/docx/sample.docx (which Playwright also imports end to
// end) or a second committed fixture file. document.xml relationships,
// when any external ones (hyperlinks) are needed, come from docRels.
func buildDocx(t *testing.T, stylesXML, bodyXML string) []byte {
	t.Helper()
	return buildDocxFull(t, stylesXML, "", "", bodyXML, nil)
}

// buildDocxWithNumbering is buildDocx plus a numbering.xml part, for the
// list-related tests.
func buildDocxWithNumbering(t *testing.T, stylesXML, numberingXML, bodyXML string) []byte {
	t.Helper()
	return buildDocxFull(t, stylesXML, numberingXML, "", bodyXML, nil)
}

// buildDocxWithHyperlinkRels is buildDocx plus a word/_rels/document.xml.rels
// part declaring external hyperlink relationships, for link tests that
// need a real w:hyperlink r:id to resolve against.
func buildDocxWithHyperlinkRels(t *testing.T, bodyXML string, hyperlinks map[string]string) []byte {
	t.Helper()
	var rels strings.Builder
	rels.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	rels.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + "\n")
	for id, target := range hyperlinks {
		rels.WriteString(`<Relationship Id="` + id + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="` + target + `" TargetMode="External"/>` + "\n")
	}
	rels.WriteString(`</Relationships>`)
	return buildDocxFull(t, "", "", rels.String(), bodyXML, nil)
}

// buildDocxWithImageRel is buildDocx plus one image-type relationship
// (id "rIdImg") in word/_rels/document.xml.rels, and, unless external,
// the referenced bytes written to word/media/image1.dat (an arbitrary
// extension, since Convert sniffs content rather than trusting names).
func buildDocxWithImageRel(t *testing.T, bodyXML string, imageBytes []byte, external bool) []byte {
	t.Helper()
	const target = "media/image1.dat"
	rel := `<Relationship Id="rIdImg" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="`
	media := map[string]string{}
	if external {
		rel += "https://example.test/pic.dat" + `" TargetMode="External"/>`
	} else {
		rel += target + `"/>`
		media["word/"+target] = string(imageBytes)
	}
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rel + `</Relationships>`
	return buildDocxFull(t, "", "", docRels, bodyXML, media)
}

func buildDocxFull(t *testing.T, stylesXML, numberingXML, docRelsXML, bodyXML string, mediaFiles map[string]string) []byte {
	t.Helper()

	overrides := ""
	if stylesXML != "" {
		overrides += `<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>`
	}
	if numberingXML != "" {
		overrides += `<Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>`
	}
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
` + overrides + `
</Types>`

	const rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<w:body>` + bodyXML + `</w:body>
</w:document>`

	type fileEntry struct{ name, data string }
	files := []fileEntry{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rootRels},
		{"word/document.xml", document},
	}
	if stylesXML != "" {
		files = append(files, fileEntry{"word/styles.xml", stylesXML})
	}
	if numberingXML != "" {
		files = append(files, fileEntry{"word/numbering.xml", numberingXML})
	}
	if docRelsXML != "" {
		files = append(files, fileEntry{"word/_rels/document.xml.rels", docRelsXML})
	}
	for name, data := range mediaFiles {
		files = append(files, fileEntry{name, data})
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			t.Fatalf("zip create %s: %v", f.name, err)
		}
		if _, err := w.Write([]byte(f.data)); err != nil {
			t.Fatalf("zip write %s: %v", f.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

// buildMinimalDocxNoTitle exercises Convert's part-discovery (via
// _rels/.rels, not a fixed path): no styles.xml, no Title, no heading.
func buildMinimalDocxNoTitle(t *testing.T, paragraphText string) []byte {
	t.Helper()
	body := `<w:p><w:r><w:t xml:space="preserve">` + paragraphText + `</w:t></w:r></w:p>`
	return buildDocx(t, "", body)
}

// TestConvertSampleDocxRunMarksAndTrackedChanges covers the run-level
// mapping rules (SPEC B4): bold -> strong, italic -> em, a run split
// mid-word still reading as one word, tracked inserts kept and deletes
// dropped, and field instruction text (the legacy HYPERLINK field's own
// w:instrText) never leaking into the visible text.
func TestConvertSampleDocxRunMarksAndTrackedChanges(t *testing.T) {
	result, err := convertFixture(t, "sample.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}

	for _, want := range []string{
		"Ask <strong>Sam</strong> before ordering more <em>toner</em>.",
		"<p>A run can be split midword and still read as one word.</p>",
		"Tracked changes: please order two boxes",
	} {
		if !strings.Contains(result.HTML, want) {
			t.Errorf("HTML missing %q; got:\n%s", want, result.HTML)
		}
	}
	if strings.Contains(result.HTML, "order one box") {
		t.Error("deleted text (w:del/w:delText) should never appear")
	}
	if strings.Contains(result.HTML, "HYPERLINK") {
		t.Error("field instruction text (w:instrText) should never appear")
	}
	if strings.Contains(result.HTML, "<strong>This text is explicitly not bold") {
		t.Error("w:b val=0 should not render <strong>, even with no ambient style bold")
	}
}

// TestConvertSampleDocxCountsNotes covers SPEC B4: "left out, and counted
// in notes: comments, footnotes and endnotes, headers and footers" plus
// P1-32's unsupported items (the fixture's EMF picture, chart, textless
// shape and equation) — SmartArt is deliberately not among them, since
// its mc:Choice branch is skipped in favour of mc:Fallback, which here
// happens to be plain text rather than another unsupported construct.
func TestConvertSampleDocxCountsNotes(t *testing.T) {
	result, err := convertFixture(t, "sample.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := []string{
		"the header", "the footer", "a comment", "a footnote",
		"a picture", "a chart", "a shape", "an equation",
	}
	if len(result.Notes) != len(want) {
		t.Fatalf("Notes = %v, want %v", result.Notes, want)
	}
	for _, w := range want {
		found := false
		for _, n := range result.Notes {
			if n == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Notes %v missing %q", result.Notes, w)
		}
	}
}

func TestConvertEndnoteReferenceCountsAsANote(t *testing.T) {
	body := `<w:p><w:r><w:t xml:space="preserve">See note.</w:t></w:r><w:r><w:endnoteReference w:id="1"/></w:r></w:p>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(result.Notes) != 1 || result.Notes[0] != "an endnote" {
		t.Errorf("Notes = %v, want [\"an endnote\"]", result.Notes)
	}
}

// TestConvertBoldOffOverridesABoldStyle covers SPEC B4's "honouring
// w:val=0/false and style inheritance": a paragraph style whose own rPr
// makes its runs bold by default, with one run overriding that off
// directly — the override must win, and an ordinary run in the same
// paragraph must still inherit the style's bold.
func TestConvertBoldOffOverridesABoldStyle(t *testing.T) {
	const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
<w:style w:type="paragraph" w:styleId="Warn">
<w:name w:val="Warning"/>
<w:basedOn w:val="Normal"/>
<w:rPr><w:b/></w:rPr>
</w:style>
</w:styles>`
	body := `<w:p><w:pPr><w:pStyle w:val="Warn"/></w:pPr>` +
		`<w:r><w:t xml:space="preserve">bold from the style</w:t></w:r>` +
		`<w:r><w:rPr><w:b w:val="0"/></w:rPr><w:t xml:space="preserve">not bold, overridden</w:t></w:r>` +
		`</w:p>`
	data := buildDocx(t, styles, body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<p><strong>bold from the style</strong>not bold, overridden</p>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertBasedOnChainResolvesHeadingLevel covers SPEC B4's basedOn
// chain resolution: a style two hops from the one that actually sets
// w:outlineLvl must still resolve to the right heading level.
func TestConvertBasedOnChainResolvesHeadingLevel(t *testing.T) {
	const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/></w:style>
<w:style w:type="paragraph" w:styleId="H1Base">
<w:name w:val="heading 1"/>
<w:basedOn w:val="Normal"/>
<w:pPr><w:outlineLvl w:val="0"/></w:pPr>
</w:style>
<w:style w:type="paragraph" w:styleId="Custom">
<w:name w:val="Custom Heading"/>
<w:basedOn w:val="H1Base"/>
</w:style>
</w:styles>`
	body := `<w:p><w:pPr><w:pStyle w:val="Title"/></w:pPr><w:r><w:t xml:space="preserve">Doc Title</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:pStyle w:val="Custom"/></w:pPr><w:r><w:t xml:space="preserve">Section title</w:t></w:r></w:p>`
	data := buildDocx(t, styles, body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.Contains(result.HTML, "<h2>Section title</h2>") {
		t.Errorf("HTML missing %q (heading level resolved through a 2-hop basedOn chain); got:\n%s", "<h2>Section title</h2>", result.HTML)
	}
}

// TestConvertTrackedInsertAndDelete is the narrow, isolated version of
// the tracked-changes check in TestConvertSampleDocxRunMarksAndTrackedChanges.
func TestConvertTrackedInsertAndDelete(t *testing.T) {
	body := `<w:p>` +
		`<w:r><w:t xml:space="preserve">Please order </w:t></w:r>` +
		`<w:ins w:id="1" w:author="Sam" w:date="2026-01-01T00:00:00Z"><w:r><w:t xml:space="preserve">two boxes</w:t></w:r></w:ins>` +
		`<w:del w:id="2" w:author="Sam" w:date="2026-01-01T00:00:00Z"><w:r><w:delText xml:space="preserve">one box</w:delText></w:r></w:del>` +
		`<w:r><w:t xml:space="preserve">.</w:t></w:r>` +
		`</w:p>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<p>Please order two boxes.</p>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertDescendsIntoStructuredDocumentTags covers SPEC B4: "descend
// into w:sdt/w:sdtContent" — a paragraph wrapped entirely in an sdt (a
// content control, common in templated Word documents) must still be
// found and converted normally.
func TestConvertDescendsIntoStructuredDocumentTags(t *testing.T) {
	body := `<w:sdt><w:sdtPr><w:alias w:val="Ignored"/></w:sdtPr><w:sdtContent>` +
		`<w:p><w:r><w:t xml:space="preserve">Inside an sdt.</w:t></w:r></w:p>` +
		`</w:sdtContent></w:sdt>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<p>Inside an sdt.</p>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertTitleFromFirstHeading1 covers SPEC B4's title fallback: no
// Title style anywhere, so the first non-empty heading 1 paragraph
// becomes the title and is removed from the body, leaving everything
// else (including text before it) in place.
func TestConvertTitleFromFirstHeading1(t *testing.T) {
	const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
<w:style w:type="paragraph" w:styleId="H1">
<w:name w:val="heading 1"/>
<w:basedOn w:val="Normal"/>
<w:pPr><w:outlineLvl w:val="0"/></w:pPr>
</w:style>
</w:styles>`
	body := `<w:p><w:r><w:t xml:space="preserve">Just some intro text, not a title.</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:pStyle w:val="H1"/></w:pPr><w:r><w:t xml:space="preserve">Getting Started</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t xml:space="preserve">Body text.</w:t></w:r></w:p>`
	data := buildDocx(t, styles, body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if result.Title != "Getting Started" {
		t.Errorf("Title = %q, want %q", result.Title, "Getting Started")
	}
	if strings.Contains(result.HTML, "Getting Started") {
		t.Error("title text should be removed from the body HTML")
	}
	if !strings.Contains(result.HTML, "Just some intro text") {
		t.Error("the paragraph before the heading 1 should remain in the body")
	}
}

// TestConvertTabToSpaceAndBreaksHandledPerType covers SPEC B4: "w:tab ->
// space; w:br -> br, with page and column breaks ignored."
func TestConvertTabToSpaceAndBreaksHandledPerType(t *testing.T) {
	body := `<w:p>` +
		`<w:r><w:t xml:space="preserve">Before</w:t></w:r>` +
		`<w:r><w:tab/></w:r>` +
		`<w:r><w:t xml:space="preserve">After</w:t></w:r>` +
		`<w:r><w:br/></w:r>` +
		`<w:r><w:t xml:space="preserve">NewLine</w:t></w:r>` +
		`<w:r><w:br w:type="page"/></w:r>` +
		`<w:r><w:br w:type="column"/></w:r>` +
		`<w:r><w:t xml:space="preserve">StillHere</w:t></w:r>` +
		`</w:p>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<p>Before After<br>NewLineStillHere</p>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

const bulletNumbering = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:abstractNum w:abstractNumId="0">
<w:lvl w:ilvl="0"><w:numFmt w:val="bullet"/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>
</w:numbering>`

const threeLevelNumbering = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:abstractNum w:abstractNumId="0">
<w:lvl w:ilvl="0"><w:numFmt w:val="bullet"/></w:lvl>
<w:lvl w:ilvl="1"><w:numFmt w:val="decimal"/></w:lvl>
<w:lvl w:ilvl="2"><w:numFmt w:val="bullet"/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>
</w:numbering>`

func listParagraph(numID, ilvl int, text string) string {
	numPr := `<w:numPr><w:ilvl w:val="` + strconv.Itoa(ilvl) + `"/><w:numId w:val="` + strconv.Itoa(numID) + `"/></w:numPr>`
	return `<w:p><w:pPr>` + numPr + `</w:pPr><w:r><w:t xml:space="preserve">` + text + `</w:t></w:r></w:p>`
}

// TestConvertNestedThreeLevelsMixedBulletsAndNumbers covers SPEC B4:
// "ilvl -> nesting", with the format (and so the ul/ol tag) changing at
// each level, matching sample.docx's numId 1 (bullet/decimal levels)
// generalised to three levels deep.
func TestConvertNestedThreeLevelsMixedBulletsAndNumbers(t *testing.T) {
	body := listParagraph(1, 0, "Top A") +
		listParagraph(1, 1, "Mid A1") +
		listParagraph(1, 2, "Deep A1a") +
		listParagraph(1, 1, "Mid A2") +
		listParagraph(1, 0, "Top B")
	data := buildDocxWithNumbering(t, "", threeLevelNumbering, body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<ul>\n" +
		"<li>Top A<ol>\n" +
		"<li>Mid A1<ul>\n" +
		"<li>Deep A1a</li></ul>\n" +
		"</li>\n" +
		"<li>Mid A2</li></ol>\n" +
		"</li>\n" +
		"<li>Top B</li></ul>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertListInterruptedByAParagraphStartsANewList covers SPEC B4:
// "consecutive paragraphs with the same numId form one list" — an
// ordinary paragraph in between, even with no numbering of its own,
// ends the first list rather than being swallowed into it.
func TestConvertListInterruptedByAParagraphStartsANewList(t *testing.T) {
	body := listParagraph(1, 0, "First") +
		listParagraph(1, 0, "Second") +
		`<w:p><w:r><w:t xml:space="preserve">Interrupting paragraph.</w:t></w:r></w:p>` +
		listParagraph(1, 0, "Third") +
		listParagraph(1, 0, "Fourth")
	data := buildDocxWithNumbering(t, "", bulletNumbering, body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<ul>\n<li>First</li>\n<li>Second</li></ul>\n" +
		"<p>Interrupting paragraph.</p>\n" +
		"<ul>\n<li>Third</li>\n<li>Fourth</li></ul>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertListDefinedViaParagraphStyle covers SPEC B4: "w:numPr,
// directly or through the paragraph style" — no paragraph here has its
// own direct numPr; the list comes entirely from the style.
func TestConvertListDefinedViaParagraphStyle(t *testing.T) {
	const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
<w:style w:type="paragraph" w:styleId="ListStyle">
<w:name w:val="My List"/>
<w:basedOn w:val="Normal"/>
<w:pPr><w:numPr><w:numId w:val="1"/></w:numPr></w:pPr>
</w:style>
</w:styles>`
	body := `<w:p><w:pPr><w:pStyle w:val="ListStyle"/></w:pPr><w:r><w:t xml:space="preserve">Item one</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:pStyle w:val="ListStyle"/></w:pPr><w:r><w:t xml:space="preserve">Item two</w:t></w:r></w:p>`
	data := buildDocxFull(t, styles, bulletNumbering, "", body, nil)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<ul>\n<li>Item one</li>\n<li>Item two</li></ul>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertComplexFieldHyperlinkSpanningRuns covers SPEC B4: "HYPERLINK
// simple or complex fields -> a href" — the field's display portion here
// spans two runs with different formatting, which must still become one
// anchor, not two adjacent ones.
func TestConvertComplexFieldHyperlinkSpanningRuns(t *testing.T) {
	body := `<w:p>` +
		`<w:r><w:fldChar w:fldCharType="begin"/></w:r>` +
		`<w:r><w:instrText xml:space="preserve"> HYPERLINK "https://example.test/page" </w:instrText></w:r>` +
		`<w:r><w:fldChar w:fldCharType="separate"/></w:r>` +
		`<w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">Bold part</w:t></w:r>` +
		`<w:r><w:t xml:space="preserve">plain part</w:t></w:r>` +
		`<w:r><w:fldChar w:fldCharType="end"/></w:r>` +
		`</w:p>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := `<p><a href="https://example.test/page"><strong>Bold part</strong>plain part</a></p>` + "\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertMailtoLink covers SPEC B4's w:hyperlink external
// relationship path with a mailto: target.
func TestConvertMailtoLink(t *testing.T) {
	body := `<w:p><w:hyperlink r:id="rIdMail"><w:r><w:t xml:space="preserve">Email Sam</w:t></w:r></w:hyperlink></w:p>`
	data := buildDocxWithHyperlinkRels(t, body, map[string]string{"rIdMail": "mailto:sam@example.test"})

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := `<p><a href="mailto:sam@example.test">Email Sam</a></p>` + "\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertInternalAnchorKeepsTextDropsLink covers SPEC B4: "Internal
// anchors keep their text and drop the link" — a w:hyperlink with only a
// w:anchor (no r:id) has nothing external to link to.
func TestConvertInternalAnchorKeepsTextDropsLink(t *testing.T) {
	body := `<w:p><w:hyperlink w:anchor="Section2"><w:r><w:t xml:space="preserve">Jump to Section 2</w:t></w:r></w:hyperlink></w:p>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<p>Jump to Section 2</p>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

func cellPara(text string) string {
	if text == "" {
		return `<w:p/>`
	}
	return `<w:p><w:r><w:t xml:space="preserve">` + text + `</w:t></w:r></w:p>`
}

// TestConvert3x3TableWithMergesAndHeaderRow covers SPEC B4: "w:tblHeader
// rows -> th", "gridSpan -> colspan", "vMerge restart/continue -> rowspan
// computed per column" — a horizontal merge in the header row and a
// vertical merge spanning two data rows, each column's rowspan computed
// independently of the others.
func TestConvert3x3TableWithMergesAndHeaderRow(t *testing.T) {
	body := `<w:tbl>` +
		`<w:tblGrid><w:gridCol/><w:gridCol/><w:gridCol/></w:tblGrid>` +
		`<w:tr><w:trPr><w:tblHeader/></w:trPr>` +
		`<w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr>` + cellPara("A") + `</w:tc>` +
		`<w:tc>` + cellPara("B") + `</w:tc>` +
		`</w:tr>` +
		`<w:tr>` +
		`<w:tc>` + cellPara("C") + `</w:tc>` +
		`<w:tc>` + cellPara("D") + `</w:tc>` +
		`<w:tc><w:tcPr><w:vMerge w:val="restart"/></w:tcPr>` + cellPara("E") + `</w:tc>` +
		`</w:tr>` +
		`<w:tr>` +
		`<w:tc>` + cellPara("F") + `</w:tc>` +
		`<w:tc>` + cellPara("G") + `</w:tc>` +
		`<w:tc><w:tcPr><w:vMerge/></w:tcPr>` + cellPara("") + `</w:tc>` +
		`</w:tr>` +
		`</w:tbl>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<table>\n" +
		"<tr>\n" +
		`<th colspan="2"><p>A</p>` + "\n</th>\n" +
		"<th><p>B</p>\n</th>\n" +
		"</tr>\n" +
		"<tr>\n" +
		"<td><p>C</p>\n</td>\n" +
		"<td><p>D</p>\n</td>\n" +
		`<td rowspan="2"><p>E</p>` + "\n</td>\n" +
		"</tr>\n" +
		"<tr>\n" +
		"<td><p>F</p>\n</td>\n" +
		"<td><p>G</p>\n</td>\n" +
		"</tr>\n" +
		"</table>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertNestedTableFlattenedIntoCellParagraphs covers SPEC B4: "A
// table nested in a cell is flattened into paragraphs in that cell" — the
// nested table's own cell paragraphs must appear in reading order, right
// where the nested table was, not as a nested <table>.
func TestConvertNestedTableFlattenedIntoCellParagraphs(t *testing.T) {
	nestedTable := `<w:tbl>` +
		`<w:tr><w:tc>` + cellPara("Nested A") + `</w:tc></w:tr>` +
		`<w:tr><w:tc>` + cellPara("Nested B") + `</w:tc></w:tr>` +
		`</w:tbl>`
	outerTable := `<w:tbl><w:tr><w:tc>` +
		cellPara("Before nested table") + nestedTable + cellPara("After nested table") +
		`</w:tc></w:tr></w:tbl>`
	data := buildDocx(t, "", outerTable)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<table>\n<tr>\n<td>" +
		"<p>Before nested table</p>\n" +
		"<p>Nested A</p>\n" +
		"<p>Nested B</p>\n" +
		"<p>After nested table</p>\n" +
		"</td>\n</tr>\n</table>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertTextBoxInChoiceWithVMLFallbackAppearsOnce covers SPEC B4:
// "Inside mc:AlternateContent, use the one branch that yields an image or
// text, never both" — here a text box appears in both a modern
// (mc:Choice) and legacy VML (mc:Fallback) shape, and its text must come
// through exactly once, not twice.
func TestConvertTextBoxInChoiceWithVMLFallbackAppearsOnce(t *testing.T) {
	body := `<w:p><w:r><mc:AlternateContent ` +
		`xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" ` +
		`xmlns:wps="http://schemas.microsoft.com/office/word/2010/wordprocessingShape" ` +
		`xmlns:v="urn:schemas-microsoft-com:vml">` +
		`<mc:Choice Requires="wps">` +
		`<wps:txbx><w:txbxContent>` + cellPara("Reminder text") + `</w:txbxContent></wps:txbx>` +
		`</mc:Choice>` +
		`<mc:Fallback>` +
		`<w:pict><v:shape><v:textbox><w:txbxContent>` + cellPara("Reminder text") + `</w:txbxContent></v:textbox></v:shape></w:pict>` +
		`</mc:Fallback>` +
		`</mc:AlternateContent></w:r></w:p>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := "<p>Reminder text</p>\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

func tinyPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// pictureDrawing returns a <w:drawing> referencing relID as a real
// DrawingML picture (SPEC B4), with alt from wp:docPr descr, declaring
// its own namespaces so it doesn't depend on what the surrounding
// document element happens to declare.
func pictureDrawing(relID, alt string) string {
	descrAttr := ""
	if alt != "" {
		descrAttr = ` descr="` + alt + `"`
	}
	return `<w:drawing ` +
		`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
		`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
		`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<wp:inline><wp:docPr id="1" name="Pic"` + descrAttr + `/>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic><pic:blipFill><a:blip r:embed="` + relID + `"/></pic:blipFill></pic:pic>` +
		`</a:graphicData></a:graphic></wp:inline></w:drawing>`
}

// TestConvertDrawingPictureExtractsImageWithAlt covers SPEC B4: "w:drawing
// ... a:blip r:embed", with alt "from wp:docPr descr".
func TestConvertDrawingPictureExtractsImageWithAlt(t *testing.T) {
	pngBytes := tinyPNGBytes(t)
	body := `<w:p><w:r>` + pictureDrawing("rIdImg", "A test picture") + `</w:r></w:p>`
	data := buildDocxWithImageRel(t, body, pngBytes, false)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("Images = %+v, want 1", result.Images)
	}
	img := result.Images[0]
	if img.Alt != "A test picture" {
		t.Errorf("Alt = %q, want %q", img.Alt, "A test picture")
	}
	if !bytes.Equal(img.Bytes, pngBytes) {
		t.Error("image bytes don't match the source PNG")
	}
	want := `<p><img src="` + img.Token + `" alt="A test picture"></p>` + "\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertVMLPictureExtractsImage covers SPEC B4: "legacy VML
// v:imagedata r:id -> img".
func TestConvertVMLPictureExtractsImage(t *testing.T) {
	pngBytes := tinyPNGBytes(t)
	body := `<w:p><w:r><w:pict xmlns:v="urn:schemas-microsoft-com:vml">` +
		`<v:shape><v:imagedata r:id="rIdImg"/></v:shape></w:pict></w:r></w:p>`
	data := buildDocxWithImageRel(t, body, pngBytes, false)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("Images = %+v, want 1", result.Images)
	}
	want := `<p><img src="` + result.Images[0].Token + `" alt=""></p>` + "\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertExternalImageNeverFetchedBecomesPlaceholder covers SPEC B4:
// "Images with TargetMode='External' are never fetched; they become
// placeholders."
func TestConvertExternalImageNeverFetchedBecomesPlaceholder(t *testing.T) {
	body := `<w:p><w:r>` + pictureDrawing("rIdImg", "") + `</w:r></w:p>`
	data := buildDocxWithImageRel(t, body, nil, true)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(result.Images) != 0 {
		t.Errorf("Images = %+v, want none (external targets are never fetched)", result.Images)
	}
	want := `<p><div data-missing-kind="picture"></div></p>` + "\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertUnsupportedImageFormatBecomesPlaceholder covers SPEC B4:
// pictures in a format this converter doesn't recognise (standing in for
// EMF/WMF/TIFF/BMP, which sample.docx already covers with a real EMF
// signature) become a "picture" placeholder, not real image bytes.
func TestConvertUnsupportedImageFormatBecomesPlaceholder(t *testing.T) {
	body := `<w:p><w:r>` + pictureDrawing("rIdImg", "") + `</w:r></w:p>`
	data := buildDocxWithImageRel(t, body, []byte("not a real image, just some bytes"), false)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(result.Images) != 0 {
		t.Errorf("Images = %+v, want none", result.Images)
	}
	want := `<p><div data-missing-kind="picture"></div></p>` + "\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertOversizeImageBecomesPlaceholder covers SPEC B4's 20 MB
// per-image limit — real PNG bytes, so a placeholder here can only be
// because of size, not because sniffing failed.
func TestConvertOversizeImageBecomesPlaceholder(t *testing.T) {
	oversized := append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, MaxImageSize)...)
	body := `<w:p><w:r>` + pictureDrawing("rIdImg", "") + `</w:r></w:p>`
	data := buildDocxWithImageRel(t, body, oversized, false)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(result.Images) != 0 {
		t.Errorf("Images = %+v, want none (over the 20 MB per-image cap)", result.Images)
	}
}

// TestConvertRelationshipPathTraversalIgnored covers SPEC B4:
// "Relationship targets resolve only inside the zip" — a target that
// climbs above the relationship file's own directory must never resolve
// to some unrelated entry the zip happens to contain, even a real image
// deliberately placed where a naive resolution would find it.
func TestConvertRelationshipPathTraversalIgnored(t *testing.T) {
	pngBytes := tinyPNGBytes(t)
	body := `<w:p><w:r>` + pictureDrawing("rIdImg", "") + `</w:r></w:p>`
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rIdImg" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../../x"/>` +
		`</Relationships>`
	data := buildDocxFull(t, "", "", docRels, body, map[string]string{"x": string(pngBytes)})

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(result.Images) != 0 {
		t.Errorf("Images = %+v, want none (path traversal target must be ignored)", result.Images)
	}
	want := `<p><div data-missing-kind="picture"></div></p>` + "\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertDiagramGraphicDataBecomesPlaceholder covers SPEC B4's
// SmartArt (dgm:) placeholder, via a bare w:drawing (not wrapped in
// mc:AlternateContent, unlike sample.docx's own SmartArt, which this
// converter never even inspects since it always prefers mc:Fallback).
func TestConvertDiagramGraphicDataBecomesPlaceholder(t *testing.T) {
	body := `<w:p><w:r><w:drawing xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
		`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><wp:inline>` +
		`<wp:docPr id="1" name="Diagram"/>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/diagram"/></a:graphic>` +
		`</wp:inline></w:drawing></w:r></w:p>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := `<p><div data-missing-kind="diagram"></div></p>` + "\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// TestConvertObjectWithCachedImageUsedAsPicture covers SPEC B4: an
// embedded object's own VML cached preview is used as a real picture
// when present.
func TestConvertObjectWithCachedImageUsedAsPicture(t *testing.T) {
	pngBytes := tinyPNGBytes(t)
	body := `<w:p><w:r><w:object><v:shape xmlns:v="urn:schemas-microsoft-com:vml">` +
		`<v:imagedata r:id="rIdImg"/></v:shape></w:object></w:r></w:p>`
	data := buildDocxWithImageRel(t, body, pngBytes, false)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("Images = %+v, want 1 (the object's own cached preview image)", result.Images)
	}
}

// TestConvertObjectWithoutImageBecomesPlaceholder covers SPEC B4:
// "embedded objects with no usable image" become an "object" placeholder.
func TestConvertObjectWithoutImageBecomesPlaceholder(t *testing.T) {
	body := `<w:p><w:r><w:object><v:shape xmlns:v="urn:schemas-microsoft-com:vml"><v:path/></v:shape></w:object></w:r></w:p>`
	data := buildDocx(t, "", body)

	result, err := Convert(bytes.NewReader(data), int64(len(data)), "test.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := `<p><div data-missing-kind="object"></div></p>` + "\n"
	if result.HTML != want {
		t.Errorf("HTML = %q, want %q", result.HTML, want)
	}
}

// FuzzConvert seeds from every committed fixture (real documents and the
// bad/ ones alike) and asserts only that Convert never panics — it may
// return any error, or a Result, but must always return cleanly. Run
// locally with `go test -fuzz=FuzzConvert -fuzztime=30s` before adding
// any new seed corpus entries by hand; this form (no -fuzz flag) just
// replays the seed corpus as ordinary subtests, which is what CI does.
func FuzzConvert(f *testing.F) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		f.Fatal("could not determine test file path")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "e2e", "fixtures", "docx")

	for _, name := range []string{
		"sample.docx", "googledocs.docx", "big-20pages.docx",
		"bad/oldword.doc", "bad/protected.docx", "bad/file.pdf",
		"bad/truncated.docx", "bad/zipbomb.docx", "bad/deep.docx",
	} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			f.Fatalf("read fixture %s: %v", name, err)
		}
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Convert panicked: %v", r)
			}
		}()
		_, _ = Convert(bytes.NewReader(data), int64(len(data)), "fuzz.docx")
	})
}
