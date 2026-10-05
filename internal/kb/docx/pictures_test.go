package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// SPEC gates 7.60-7.68 (B13.9, D-99): every picture from Word.

// pictureColour mirrors e2e/fixtures/docx/gen/pictures.go.
func pictureColour(n int) color.RGBA {
	return color.RGBA{R: uint8(n * 9 % 256), G: uint8(255 - n*7%256), B: uint8(n * 31 % 256), A: 255}
}

func colourOf(t *testing.T, b []byte) color.RGBA {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	r, g, bl, a := img.At(0, 0).RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8), A: uint8(a >> 8)}
}

var imgSrcRe = regexp.MustCompile(`<img src="([^"]*)"`)

func imgTokens(html string) []string {
	var out []string
	for _, m := range imgSrcRe.FindAllStringSubmatch(html, -1) {
		out = append(out, m[1])
	}
	return out
}

func imageByToken(t *testing.T, r Result, token string) Image {
	t.Helper()
	for _, im := range r.Images {
		if im.Token == token {
			return im
		}
	}
	t.Fatalf("no image carries the token %q", token)
	return Image{}
}

// Gate 7.60: twenty-five different pictures, each the right one, in the right
// place. At v1.3.0 the token for picture 1 was a prefix of the tokens for
// pictures 10-19, so replacing it corrupted them.
func TestTwentyFivePicturesEachKeepTheirOwnToken(t *testing.T) {
	r, err := convertFixture(t, "pictures-25.docx")
	if err != nil {
		t.Fatal(err)
	}
	tokens := imgTokens(r.HTML)
	if len(tokens) != 25 || len(r.Images) != 25 {
		t.Fatalf("%d <img> and %d images, want 25 of each", len(tokens), len(r.Images))
	}
	for i, tok := range tokens {
		for j, other := range tokens {
			if i != j && strings.Contains(other, tok) {
				t.Errorf("token %d (%q) is contained in token %d (%q): replacing one would corrupt the other", i+1, tok, j+1, other)
			}
		}
		got := colourOf(t, imageByToken(t, r, tok).Bytes)
		if want := pictureColour(i + 1); got != want {
			t.Errorf("picture %d in the document is colour %v, want %v (the wrong picture, or in the wrong place)", i+1, got, want)
		}
	}
	if strings.Contains(r.HTML, `data-missing-kind`) {
		t.Errorf("a placeholder appeared where every picture is fine: %s", r.HTML)
	}
}

// Gate 7.62: pictures in documents made by other programs, whose targets are
// absolute, percent-encoded, or spelt in a different case from the part.
func TestPicturesWithUnusualTargetsComeAcross(t *testing.T) {
	r, err := convertFixture(t, "pictures-oddnames.docx")
	if err != nil {
		t.Fatal(err)
	}
	tokens := imgTokens(r.HTML)
	if len(tokens) != 3 || strings.Contains(r.HTML, "data-missing-kind") {
		t.Fatalf("html = %s, want three pictures and no placeholder", r.HTML)
	}
	for i, tok := range tokens {
		if got, want := colourOf(t, imageByToken(t, r, tok).Bytes), pictureColour(i+1); got != want {
			t.Errorf("picture %d is colour %v, want %v", i+1, got, want)
		}
	}
}

// Gate 7.63: a picture in the title's own paragraph is kept, at the top of the article.
func TestPictureInTheTitleParagraphIsKeptAtTheTop(t *testing.T) {
	r, err := convertFixture(t, "picture-in-title.docx")
	if err != nil {
		t.Fatal(err)
	}
	if r.Title != "Quarterly report" {
		t.Errorf("Title = %q", r.Title)
	}
	tokens := imgTokens(r.HTML)
	if len(tokens) != 2 {
		t.Fatalf("%d pictures, want the logo and the body picture; html = %s", len(tokens), r.HTML)
	}
	if first := strings.Index(r.HTML, "<img"); first > strings.Index(r.HTML, "Some words") {
		t.Errorf("the title's picture is not at the top: %s", r.HTML)
	}
	if got := colourOf(t, imageByToken(t, r, tokens[0]).Bytes); got != pictureColour(1) {
		t.Errorf("first picture is %v, want the logo %v", got, pictureColour(1))
	}
}

// Gate 7.64 (the converter's half): a picture it cannot use is a marked
// placeholder, counted in Notes; the handler's half (a failed save) is in
// import_docx_integration_test.go.

// Gate 7.67: every picture the document refers to is either shown or marked in
// place. The count is read from document.xml itself, independently of the
// converter, so a picture that goes missing without a mark fails here.
func referencedPictureCount(t *testing.T, docxPath string) int {
	t.Helper()
	data, err := os.ReadFile(docxPath)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var doc []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			doc, _ = io.ReadAll(rc)
			rc.Close()
		}
	}
	if doc == nil {
		t.Fatalf("%s has no word/document.xml", docxPath)
	}

	type alt struct {
		choice, fallback int
		branch           string // "", "choice" or "fallback"
	}
	var stack []*alt
	total := 0
	count := func() {
		if len(stack) == 0 {
			total++
			return
		}
		top := stack[len(stack)-1]
		if top.branch == "choice" {
			top.choice++
		} else {
			top.fallback++
		}
	}
	dec := xml.NewDecoder(bytes.NewReader(doc))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch e := tok.(type) {
		case xml.StartElement:
			switch e.Name.Local {
			case "AlternateContent":
				stack = append(stack, &alt{})
			case "Choice":
				if len(stack) > 0 {
					stack[len(stack)-1].branch = "choice"
				}
			case "Fallback":
				if len(stack) > 0 {
					stack[len(stack)-1].branch = "fallback"
				}
			case "blip":
				for _, a := range e.Attr {
					if a.Name.Local == "embed" && a.Value != "" {
						count()
					}
				}
			case "imagedata":
				for _, a := range e.Attr {
					if a.Name.Local == "id" && a.Value != "" {
						count()
					}
				}
			}
		case xml.EndElement:
			if e.Name.Local == "AlternateContent" && len(stack) > 0 {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				total += max(top.choice, top.fallback)
			}
		}
	}
	return total
}

func TestEveryPictureIsShownOrMarkedInPlace(t *testing.T) {
	dir := fixturesDir(t)
	paths, _ := filepath.Glob(filepath.Join(dir, "*.docx"))
	samples, _ := filepath.Glob(filepath.Join(dir, "..", "..", "..", "samples", "word", "*.docx"))
	paths = append(paths, samples...)
	if len(paths) < 5 {
		t.Fatalf("found only %d documents to check", len(paths))
	}
	for _, path := range paths {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			r, err := Convert(bytes.NewReader(data), int64(len(data)), name)
			if err != nil {
				t.Fatal(err)
			}
			want := referencedPictureCount(t, path)
			shown := len(imgTokens(r.HTML))
			marked := strings.Count(r.HTML, `data-missing-kind="picture"`) + strings.Count(r.HTML, `data-missing-kind="drawing"`)
			if shown+marked != want {
				t.Errorf("%s refers to %d pictures, but %d are shown and %d marked in place (%d unaccounted for)",
					name, want, shown, marked, want-shown-marked)
			}
		})
	}
}
