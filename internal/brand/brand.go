// Package brand holds what makes one installation look like Comp HQ and
// another like MC HQ (D-103): the name people see, the word over HQ in the
// wordmark, the icon set, and the key theme.css uses to swap in that
// department's colours. It is chosen once, when the app starts, from
// COMPHQ_BRAND; everything else about the app is the same for both.
package brand

import (
	"fmt"
	"net/url"
	"strings"
)

// Brand is one department's face.
type Brand struct {
	// Key is the value COMPHQ_BRAND takes, and the data-brand attribute on
	// every page's <html>, which theme.css reads to pick the colours.
	Key string
	// Name is the app's name as people see it: "Comp HQ".
	Name string
	// Word is the small word over the large HQ in the wordmark: "Comp".
	Word string
	// Department names whose app it is, for the installed app's description.
	Department string
	// IconDir is where this brand's icon files live, under /static/.
	IconDir string
	// IconFile is the multi-size .ico inside IconDir.
	IconFile string
	// Manifest is the address of this brand's web app manifest.
	Manifest string
	// FilePrefix starts the names of files people download (exports).
	FilePrefix string
	// Ink is the dark brand colour (theme.css --color-ink-navy for this
	// brand), which the browser paints an installed app's window frame.
	Ink string
	// Link is the link colour in an exported article file, which is a
	// standalone page that can't read theme.css.
	Link string
}

// Comp is the original: navy and orange.
var Comp = Brand{
	Key:        "comp",
	Name:       "Comp HQ",
	Word:       "Comp",
	Department: "Comp Department",
	IconDir:    "/static/theme/icons",
	IconFile:   "comphq.ico",
	Manifest:   "/static/app/manifest.webmanifest",
	FilePrefix: "comphq",
	Ink:        "#141b2d",
	Link:       "#c2410c",
}

// MC is the MC Department's copy: crimson and gold.
var MC = Brand{
	Key:        "mc",
	Name:       "MC HQ",
	Word:       "MC",
	Department: "MC Department",
	IconDir:    "/static/theme/icons/mc",
	IconFile:   "mchq.ico",
	Manifest:   "/static/app/manifest-mc.webmanifest",
	FilePrefix: "mchq",
	Ink:        "#6b0f24",
	Link:       "#835400",
}

var all = []Brand{Comp, MC}

var current = Comp

// Current is the brand this installation runs as.
func Current() Brand { return current }

// Set chooses the brand by its key ("comp" or "mc", any case). An empty key
// means Comp HQ, so an installation that never set COMPHQ_BRAND is
// unchanged. An unknown key is an error, so a typo in the TrueNAS settings
// stops the app with a clear message rather than quietly showing the wrong
// department's name.
func Set(key string) error {
	b, err := Lookup(key)
	if err != nil {
		return err
	}
	current = b
	return nil
}

// Lookup finds a brand by its key without choosing it.
func Lookup(key string) (Brand, error) {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return Comp, nil
	}
	for _, b := range all {
		if b.Key == k {
			return b, nil
		}
	}
	keys := make([]string, len(all))
	for i, b := range all {
		keys[i] = `"` + b.Key + `"`
	}
	return Brand{}, fmt.Errorf("COMPHQ_BRAND is %q; it must be one of %s", key, strings.Join(keys, ", "))
}

// IconPath is the path, inside the embedded web/static, of this brand's
// .ico, for the handlers that serve it as a file.
func (b Brand) IconPath() string {
	return "web" + b.IconDir + "/" + b.IconFile
}

// IconDownloadName is what the downloaded icon is called, so Windows'
// Change Icon dialog shows it as this app's: "Comp HQ.ico".
func (b Brand) IconDownloadName() string { return b.Name + ".ico" }

// IconDownloadDisposition is the Content-Disposition for that download,
// with the plain name and its UTF-8 spelling.
func (b Brand) IconDownloadDisposition() string {
	name := b.IconDownloadName()
	return `attachment; filename="` + name + `"; filename*=UTF-8''` + url.PathEscape(name)
}
