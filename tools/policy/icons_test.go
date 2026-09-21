package policy

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The Comp HQ icon (SPEC B11.2, gates 5.20 and 5.21): drawn once as SVG,
// rendered to every size by tools/icons/render.mjs, stored inside the app.
// These tests hold every file to the size it claims, and the mark to the
// shape SPEC gives it, so a size can't quietly drift or go missing.

var (
	// pngSizes are the square PNGs of gate 5.21, named icon-<size>.png.
	pngSizes = []int{16, 32, 48, 64, 128, 192, 256, 512}
	// icoSizes are the sizes packed into comphq.ico.
	icoSizes = []int{16, 32, 48, 64, 128, 256}
)

const maskableName = "icon-maskable-512.png"

func iconsDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "web", "static", "theme", "icons")
}

func decodePNG(t *testing.T, data []byte, what string) image.Image {
	t.Helper()
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("%s is not a readable image: %v", what, err)
	}
	if format != "png" {
		t.Fatalf("%s is a %s, want png", what, format)
	}
	return img
}

func loadPNG(t *testing.T, name string) image.Image {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(iconsDir(t), name))
	if err != nil {
		t.Fatalf("%s is missing: %v", name, err)
	}
	return decodePNG(t, data, name)
}

func TestEveryIconPNGExistsAtItsDeclaredSize(t *testing.T) {
	for _, size := range pngSizes {
		name := "icon-" + strconv.Itoa(size) + ".png"
		b := loadPNG(t, name).Bounds()
		if b.Dx() != size || b.Dy() != size {
			t.Errorf("%s is %dx%d, want %dx%d", name, b.Dx(), b.Dy(), size, size)
		}
	}
	if b := loadPNG(t, maskableName).Bounds(); b.Dx() != 512 || b.Dy() != 512 {
		t.Errorf("%s is %dx%d, want 512x512", maskableName, b.Dx(), b.Dy())
	}
}

// The tool that draws the sizes and this test must agree on the list, or one
// of them is out of date.
func TestIconSizesMatchTheRenderer(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "tools", "icons", "render.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string][]int{"PNG_SIZES": pngSizes, "ICO_SIZES": icoSizes} {
		m := regexp.MustCompile(`export const ` + name + ` = \[([0-9, ]+)\]`).FindSubmatch(src)
		if m == nil {
			t.Fatalf("render.mjs declares no %s", name)
		}
		var got []int
		for _, f := range strings.Split(string(m[1]), ",") {
			n, err := strconv.Atoi(strings.TrimSpace(f))
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, n)
		}
		if len(got) != len(want) {
			t.Fatalf("render.mjs %s = %v, this test expects %v", name, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("render.mjs %s = %v, this test expects %v", name, got, want)
				break
			}
		}
	}
}

// comphq.ico is a real multi-size icon: a directory whose every entry names
// the size of the PNG it points at.
func TestIcoHoldsEverySizeAndEachIsAsClaimed(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(iconsDir(t), "comphq.ico"))
	if err != nil {
		t.Fatalf("comphq.ico is missing: %v", err)
	}
	if len(data) < 6 || binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		t.Fatal("comphq.ico doesn't start with the icon header (reserved 0, type 1)")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count != len(icoSizes) {
		t.Fatalf("comphq.ico holds %d images, want %d", count, len(icoSizes))
	}
	for i := 0; i < count; i++ {
		entry := data[6+16*i : 6+16*(i+1)]
		claimed := int(entry[0])
		if claimed == 0 {
			claimed = 256
		}
		if claimed != icoSizes[i] {
			t.Errorf("entry %d claims %dpx, want %dpx", i, claimed, icoSizes[i])
		}
		length := int(binary.LittleEndian.Uint32(entry[8:12]))
		offset := int(binary.LittleEndian.Uint32(entry[12:16]))
		if offset < 0 || length <= 0 || offset+length > len(data) {
			t.Fatalf("entry %d points outside the file", i)
		}
		img := decodePNG(t, data[offset:offset+length], "comphq.ico entry "+strconv.Itoa(i))
		if b := img.Bounds(); b.Dx() != claimed || b.Dy() != claimed {
			t.Errorf("comphq.ico entry %d claims %dpx but its image is %dx%d", i, claimed, b.Dx(), b.Dy())
		}
	}
}

// The SVG is what everything is drawn from and draws alone: the font it needs
// is inside it, and that font is exactly the vendored one (so it can't drift),
// and nothing in it is fetched from the internet (gates 4.12, 5.20).
func TestSVGSourceIsSelfContained(t *testing.T) {
	svgBytes, err := os.ReadFile(filepath.Join(iconsDir(t), "comphq.svg"))
	if err != nil {
		t.Fatalf("comphq.svg is missing: %v", err)
	}
	svg := string(svgBytes)
	if !strings.Contains(svg, `viewBox="0 0 512 512"`) {
		t.Error("comphq.svg isn't drawn on a 512 x 512 square")
	}
	if regexp.MustCompile(`(?:src|href)=["']?https?:`).MatchString(svg) || strings.Contains(svg, "url(http") {
		t.Error("comphq.svg fetches something from the internet")
	}
	m := regexp.MustCompile(`url\(data:font/woff2;base64,([A-Za-z0-9+/=]+)\)`).FindStringSubmatch(svg)
	if m == nil {
		t.Fatal("comphq.svg doesn't embed its font")
	}
	embedded, err := base64.StdEncoding.DecodeString(m[1])
	if err != nil {
		t.Fatalf("the embedded font isn't valid base64: %v", err)
	}
	vendored, err := os.ReadFile(filepath.Join(repoRoot(t), "web", "static", "theme", "fonts", "big-shoulders-display-variable.woff2"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(embedded, vendored) {
		t.Error("the font inside comphq.svg differs from the vendored Big Shoulders Display; run `npm run icons`")
	}
	for _, id := range []string{`id="field"`, `id="stacked"`, `id="hq-only"`} {
		if !strings.Contains(svg, id) {
			t.Errorf("comphq.svg has no %s group", id)
		}
	}
}

func rgba(c color.Color) (r, g, b, a uint8) {
	R, G, B, A := c.RGBA()
	return uint8(R >> 8), uint8(G >> 8), uint8(B >> 8), uint8(A >> 8)
}

// The mark is the sidebar's: an ink-navy field with the accent-orange HQ (SPEC
// gate 5.20), at 16 px as much as at 512.
func TestIconIsNavyWithOrangeHQAtEverySize(t *testing.T) {
	for _, size := range pngSizes {
		img := loadPNG(t, "icon-"+strconv.Itoa(size)+".png")
		navy, orange := 0, 0
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, a := rgba(img.At(x, y))
				if a == 255 && r == 0x14 && g == 0x1b && bl == 0x2d {
					navy++
				}
				if a == 255 && r > 0xf0 && g > 0x60 && g < 0x78 && bl < 0x30 {
					orange++
				}
			}
		}
		total := size * size
		if navy < total/4 {
			t.Errorf("icon-%d.png is only %d%% ink navy; the field is missing", size, navy*100/total)
		}
		if orange < total/20 {
			t.Errorf("icon-%d.png has almost no accent orange (%d pixels); HQ is missing", size, orange)
		}
	}
}

// The regular icon is a rounded square (its corner is see-through); the
// maskable one fills the square, and keeps every letter inside the safe
// circle so the system can crop it to any shape without cutting one (gate 5.21).
func TestMaskableIconFillsTheSquareAndKeepsTheMarkInTheSafeCircle(t *testing.T) {
	regular := loadPNG(t, "icon-512.png")
	if _, _, _, a := rgba(regular.At(0, 0)); a != 0 {
		t.Error("icon-512.png's corner is filled; the regular icon is a rounded square")
	}

	m := loadPNG(t, maskableName)
	for _, p := range [][2]int{{0, 0}, {511, 0}, {0, 511}, {511, 511}} {
		r, g, b, a := rgba(m.At(p[0], p[1]))
		if a != 255 || r != 0x14 || g != 0x1b || b != 0x2d {
			t.Errorf("%s corner %v is (%d,%d,%d,%d), want opaque ink navy", maskableName, p, r, g, b, a)
		}
	}
	// Everything that isn't the field lies within the safe circle: 40% of the
	// side, from the centre.
	const safe = 0.4 * 512
	worst := 0.0
	for y := 0; y < 512; y++ {
		for x := 0; x < 512; x++ {
			r, g, b, _ := rgba(m.At(x, y))
			if r == 0x14 && g == 0x1b && b == 0x2d {
				continue
			}
			d := math.Hypot(float64(x)-255.5, float64(y)-255.5)
			if d > worst {
				worst = d
			}
		}
	}
	if worst > safe {
		t.Errorf("%s has mark pixels %.0fpx from the centre; the safe circle's radius is %.0fpx", maskableName, worst, safe)
	}
}

// At 16 and 32 pixels the two-line wordmark is unreadable, so those sizes
// carry HQ alone: nothing white (COMP is white) may be left in them.
func TestSmallSizesDropCOMP(t *testing.T) {
	for _, size := range []int{16, 32} {
		img := loadPNG(t, "icon-"+strconv.Itoa(size)+".png")
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, a := rgba(img.At(x, y))
				if a > 200 && r > 0xd0 && g > 0xd0 && bl > 0xd0 {
					t.Fatalf("icon-%d.png has a white pixel at (%d,%d); COMP should be dropped at this size", size, x, y)
				}
			}
		}
	}
	// And from 48 up it is there.
	found := false
	img := loadPNG(t, "icon-256.png")
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y && !found; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if r, g, bl, a := rgba(img.At(x, y)); a == 255 && r > 0xf0 && g > 0xf0 && bl > 0xf0 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("icon-256.png has no white: COMP is missing from the stacked wordmark")
	}
}
