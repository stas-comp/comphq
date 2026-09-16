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

// buildMinimalDocxNoTitle writes the smallest valid .docx package that
// exercises Convert's part-discovery (via _rels/.rels, not a fixed path):
// content types, root relationships, and a document part with one plain
// paragraph — no styles.xml, no Title, no heading.
func buildMinimalDocxNoTitle(t *testing.T, paragraphText string) []byte {
	t.Helper()

	const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

	const rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body><w:p><w:r><w:t xml:space="preserve">` + paragraphText + `</w:t></w:r></w:p></w:body>
</w:document>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range []struct{ name, data string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rootRels},
		{"word/document.xml", document},
	} {
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
