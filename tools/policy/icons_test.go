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

// The app's icon (SPEC B11.2, gates 5.20 and 5.21): drawn once as SVG,
// rendered to every size by tools/icons/render.mjs, stored inside the app.
// These tests hold every file to the size it claims, and the mark to the
// shape SPEC gives it, so a size can't quietly drift or go missing. Each
// brand (D-103) has its own set, held to the same rules.

var (
	// pngSizes are the square PNGs of gate 5.21, named icon-<size>.png.
	pngSizes = []int{16, 32, 48, 64, 128, 192, 256, 512}
	// icoSizes are the sizes packed into each brand's .ico.
	icoSizes = []int{16, 32, 48, 64, 128, 256}
)

const maskableName = "icon-maskable-512.png"

// iconBrand is one brand's icon set: the folder under
// web/static/theme/icons, the .ico and .svg names, the field colour, and a
// test for the accent colour HQ is drawn in. They match internal/brand.
type iconBrand struct {
	key   string
	dir   string // "" is the icons folder itself
	stem  string
	field [3]uint8
	isHQ  func(r, g, b uint8) bool
}

var iconBrands = []iconBrand{
	// Comp HQ: ink navy with the accent-orange HQ.
	{"comp", "", "comphq", [3]uint8{0x14, 0x1b, 0x2d}, func(r, g, b uint8) bool { return r > 0xf0 && g > 0x60 && g < 0x78 && b < 0x30 }},
	// MC HQ: crimson with the gold HQ (#f2b632).
	{"mc", "mc", "mchq", [3]uint8{0x6b, 0x0f, 0x24}, func(r, g, b uint8) bool { return r > 0xe8 && g > 0xa8 && g < 0xc0 && b < 0x48 }},
}

func (ib iconBrand) isField(r, g, b uint8) bool {
	return r == ib.field[0] && g == ib.field[1] && b == ib.field[2]
}

func forEachIconBrand(t *testing.T, f func(t *testing.T, ib iconBrand)) {
	for _, ib := range iconBrands {
		t.Run(ib.key, func(t *testing.T) { f(t, ib) })
	}
}

func iconsDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "web", "static", "theme", "icons")
}

func (ib iconBrand) path(t *testing.T, name string) string {
	return filepath.Join(iconsDir(t), ib.dir, name)
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

func (ib iconBrand) loadPNG(t *testing.T, name string) image.Image {
	t.Helper()
	data, err := os.ReadFile(ib.path(t, name))
	if err != nil {
		t.Fatalf("%s is missing: %v", name, err)
	}
	return decodePNG(t, data, name)
}

func TestEveryIconPNGExistsAtItsDeclaredSize(t *testing.T) {
	forEachIconBrand(t, func(t *testing.T, ib iconBrand) {
		for _, size := range pngSizes {
			name := "icon-" + strconv.Itoa(size) + ".png"
			b := ib.loadPNG(t, name).Bounds()
			if b.Dx() != size || b.Dy() != size {
				t.Errorf("%s is %dx%d, want %dx%d", name, b.Dx(), b.Dy(), size, size)
			}
		}
		if b := ib.loadPNG(t, maskableName).Bounds(); b.Dx() != 512 || b.Dy() != 512 {
			t.Errorf("%s is %dx%d, want 512x512", maskableName, b.Dx(), b.Dy())
		}
	})
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
	for _, ib := range iconBrands {
		if !strings.Contains(string(src), ib.key+": { design: '"+ib.stem+".design.svg'") {
			t.Errorf("render.mjs doesn't draw the %s brand from tools/icons/%s.design.svg", ib.key, ib.stem)
		}
	}
}

// Each .ico is a real multi-size icon: a directory whose every entry names
// the size of the PNG it points at.
func TestIcoHoldsEverySizeAndEachIsAsClaimed(t *testing.T) {
	forEachIconBrand(t, func(t *testing.T, ib iconBrand) {
		ico := ib.stem + ".ico"
		data, err := os.ReadFile(ib.path(t, ico))
		if err != nil {
			t.Fatalf("%s is missing: %v", ico, err)
		}
		if len(data) < 6 || binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
			t.Fatalf("%s doesn't start with the icon header (reserved 0, type 1)", ico)
		}
		count := int(binary.LittleEndian.Uint16(data[4:6]))
		if count != len(icoSizes) {
			t.Fatalf("%s holds %d images, want %d", ico, count, len(icoSizes))
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
			img := decodePNG(t, data[offset:offset+length], ico+" entry "+strconv.Itoa(i))
			if b := img.Bounds(); b.Dx() != claimed || b.Dy() != claimed {
				t.Errorf("%s entry %d claims %dpx but its image is %dx%d", ico, i, claimed, b.Dx(), b.Dy())
			}
		}
	})
}

// The SVG is what everything is drawn from and draws alone: the font it needs
// is inside it, and that font is exactly the vendored one (so it can't drift),
// and nothing in it is fetched from the internet (gates 4.12, 5.20).
func TestSVGSourceIsSelfContained(t *testing.T) {
	forEachIconBrand(t, func(t *testing.T, ib iconBrand) {
		name := ib.stem + ".svg"
		svgBytes, err := os.ReadFile(ib.path(t, name))
		if err != nil {
			t.Fatalf("%s is missing: %v", name, err)
		}
		svg := string(svgBytes)
		if !strings.Contains(svg, `viewBox="0 0 512 512"`) {
			t.Errorf("%s isn't drawn on a 512 x 512 square", name)
		}
		if regexp.MustCompile(`(?:src|href)=["']?https?:`).MatchString(svg) || strings.Contains(svg, "url(http") {
			t.Errorf("%s fetches something from the internet", name)
		}
		m := regexp.MustCompile(`url\(data:font/woff2;base64,([A-Za-z0-9+/=]+)\)`).FindStringSubmatch(svg)
		if m == nil {
			t.Fatalf("%s doesn't embed its font", name)
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
			t.Errorf("the font inside %s differs from the vendored Big Shoulders Display; run `npm run icons`", name)
		}
		for _, id := range []string{`id="field"`, `id="stacked"`, `id="hq-only"`} {
			if !strings.Contains(svg, id) {
				t.Errorf("%s has no %s group", name, id)
			}
		}
	})
}

func rgba(c color.Color) (r, g, b, a uint8) {
	R, G, B, A := c.RGBA()
	return uint8(R >> 8), uint8(G >> 8), uint8(B >> 8), uint8(A >> 8)
}

// The mark is the sidebar's: the brand's dark field with HQ in its accent
// (SPEC gate 5.20), at 16 px as much as at 512.
func TestIconIsFieldWithAccentHQAtEverySize(t *testing.T) {
	forEachIconBrand(t, func(t *testing.T, ib iconBrand) {
		for _, size := range pngSizes {
			img := ib.loadPNG(t, "icon-"+strconv.Itoa(size)+".png")
			field, accent := 0, 0
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					r, g, bl, a := rgba(img.At(x, y))
					if a == 255 && ib.isField(r, g, bl) {
						field++
					}
					if a == 255 && ib.isHQ(r, g, bl) {
						accent++
					}
				}
			}
			total := size * size
			if field < total/4 {
				t.Errorf("icon-%d.png is only %d%% field colour; the field is missing", size, field*100/total)
			}
			if accent < total/20 {
				t.Errorf("icon-%d.png has almost no accent colour (%d pixels); HQ is missing", size, accent)
			}
		}
	})
}

// The regular icon is a rounded square (its corner is see-through); the
// maskable one fills the square, and keeps every letter inside the safe
// circle so the system can crop it to any shape without cutting one (gate 5.21).
func TestMaskableIconFillsTheSquareAndKeepsTheMarkInTheSafeCircle(t *testing.T) {
	forEachIconBrand(t, func(t *testing.T, ib iconBrand) {
		regular := ib.loadPNG(t, "icon-512.png")
		if _, _, _, a := rgba(regular.At(0, 0)); a != 0 {
			t.Error("icon-512.png's corner is filled; the regular icon is a rounded square")
		}

		m := ib.loadPNG(t, maskableName)
		for _, p := range [][2]int{{0, 0}, {511, 0}, {0, 511}, {511, 511}} {
			r, g, b, a := rgba(m.At(p[0], p[1]))
			if a != 255 || !ib.isField(r, g, b) {
				t.Errorf("%s corner %v is (%d,%d,%d,%d), want the opaque field colour", maskableName, p, r, g, b, a)
			}
		}
		// Everything that isn't the field lies within the safe circle: 40% of the
		// side, from the centre.
		const safe = 0.4 * 512
		worst := 0.0
		for y := 0; y < 512; y++ {
			for x := 0; x < 512; x++ {
				r, g, b, _ := rgba(m.At(x, y))
				if ib.isField(r, g, b) {
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
	})
}

// At 16 and 32 pixels the two-line wordmark is unreadable, so those sizes
// carry HQ alone: nothing white (the small word, COMP or MC, is white) may be
// left in them.
func TestSmallSizesDropTheSmallWord(t *testing.T) {
	forEachIconBrand(t, func(t *testing.T, ib iconBrand) {
		for _, size := range []int{16, 32} {
			img := ib.loadPNG(t, "icon-"+strconv.Itoa(size)+".png")
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					r, g, bl, a := rgba(img.At(x, y))
					if a > 200 && r > 0xd0 && g > 0xd0 && bl > 0xd0 {
						t.Fatalf("icon-%d.png has a white pixel at (%d,%d); the small word should be dropped at this size", size, x, y)
					}
				}
			}
		}
		// And from 48 up it is there.
		found := false
		img := ib.loadPNG(t, "icon-256.png")
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
			t.Error("icon-256.png has no white: the small word is missing from the stacked wordmark")
		}
	})
}
