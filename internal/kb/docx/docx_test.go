package docx

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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
// styles.xml — no header/footer/comments/numbering. Building it in Go
// rather than committing a binary satisfies PLAN.md's "no hand-edited
// binaries" for rules narrow enough not to need the shared, realistic
// e2e/fixtures/docx/sample.docx (which Playwright also imports end to
// end) or a second committed fixture file.
func buildDocx(t *testing.T, stylesXML, bodyXML string) []byte {
	t.Helper()

	stylesOverride := ""
	if stylesXML != "" {
		stylesOverride = `<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>`
	}
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
` + stylesOverride + `
</Types>`

	const rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
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
// in notes: comments, footnotes and endnotes, headers and footers."
func TestConvertSampleDocxCountsNotes(t *testing.T) {
	result, err := convertFixture(t, "sample.docx")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	want := []string{"the header", "the footer", "a comment", "a footnote"}
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
