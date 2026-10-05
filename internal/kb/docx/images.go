package docx

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

// MaxImageSize is the per-image limit (SPEC B4), independent of and
// smaller than the whole-document MaxUncompressedSize budget.
const MaxImageSize = 20 * 1024 * 1024

// Image is one picture Convert successfully extracted: Token appears
// literally in Result.HTML as an <img src>, for the caller to replace
// with the real /images/... URL once it has stored Bytes through the
// same image-store function as an upload (SPEC B4) — this package never
// touches disk or a database itself. The "cid:" scheme is never allowed
// through the sanitiser, so a token the caller somehow fails to replace
// is dropped rather than left as a dangling or confusing src.
type Image struct {
	Token string
	Bytes []byte
	Alt   string
}

// mediaRelationship is one image-type relationship: either a target
// inside the zip (already resolved and path-traversal-checked), or an
// external one, which SPEC B4 says to never fetch.
type mediaRelationship struct {
	target   string
	external bool
}

// resolvedImage is the outcome of actually reading and sniffing a
// mediaRelationship's bytes, computed once per relationship up front
// (docx.go's Convert) so the pure token-walk in document.go never touches
// the zip itself — it just looks a relationship id up in a map.
type resolvedImage struct {
	supported bool
	bytes     []byte
	// legacyDrawing is an EMF or WMF: a real picture, in an old Windows
	// vector format browsers can't show (gate 7.65, D-99). It is marked with
	// its own kind of placeholder rather than the general "picture" one.
	legacyDrawing bool
}

// mediaRelationships returns mainPart's image relationships (SPEC B4:
// "w:drawing ... a:blip r:embed" and "VML v:imagedata r:id").
func (p *pkgReader) mediaRelationships(mainPart string) (map[string]mediaRelationship, error) {
	data, err := p.readPartIfExists(relsPathFor(mainPart))
	if err != nil {
		return nil, err
	}
	rels := map[string]mediaRelationship{}
	if data == nil {
		return rels, nil
	}
	parsed, err := parseRelationships(data)
	if err != nil {
		return nil, err
	}
	for _, r := range parsed {
		if !hasSuffixFold(r.Type, "/image") {
			continue
		}
		if r.TargetMode == "External" {
			rels[r.ID] = mediaRelationship{external: true}
			continue
		}
		// SPEC B4: "Relationship targets resolve only inside the zip" —
		// a target like "../../x" that would climb above mainPart's own
		// directory is ignored rather than resolved to some unrelated
		// entry the zip happens to contain.
		resolved, ok := resolveRelativeTarget(mainPart, r.Target)
		if !ok {
			continue
		}
		rels[r.ID] = mediaRelationship{target: resolved}
	}
	return rels, nil
}

// resolveRelativeTarget resolves target relative to basePart's directory
// using path.Join/Clean (zip entries always use forward slashes,
// regardless of host OS), refusing anything that climbs above that
// directory's own root.
func resolveRelativeTarget(basePart, target string) (string, bool) {
	// Other programs spell targets in ways Word doesn't (gate 7.62, B13.9
	// item 5): percent-encoded ("my%20picture.png"), and absolute from the
	// package root ("/word/media/x.png"). Part names are matched without
	// regard to case by the reader (OPC part names are case-insensitive).
	if decoded, err := url.PathUnescape(target); err == nil {
		target = decoded
	}
	if strings.HasPrefix(target, "/") {
		resolved := path.Clean(strings.TrimLeft(target, "/"))
		if resolved == ".." || strings.HasPrefix(resolved, "../") || resolved == "." {
			return "", false
		}
		return resolved, true
	}
	dir := "."
	if i := strings.LastIndexByte(basePart, '/'); i >= 0 {
		dir = basePart[:i]
	}
	resolved := path.Join(dir, target)
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", false
	}
	return resolved, true
}

// resolveImage reads and sniffs one media relationship's bytes. Anything
// that isn't a plain embedded JPEG/PNG/GIF/WebP under the 20 MB cap —
// including an external target, which is never fetched — comes back
// unsupported, so the caller renders a "picture" placeholder instead of
// including it as a real image.
func (p *pkgReader) resolveImage(rel mediaRelationship) resolvedImage {
	if rel.external {
		return resolvedImage{}
	}
	data, err := p.readImagePart(rel.target)
	if err != nil || data == nil {
		return resolvedImage{}
	}
	if !sniffSupportedImage(data) {
		return resolvedImage{legacyDrawing: isEMF(data) || isWMF(data)}
	}
	return resolvedImage{supported: true, bytes: data}
}

// sniffSupportedImage identifies the image type from its content, the
// same way internal/kb/images.Store does for an upload, so this
// converter and the actual image store agree on which formats it's
// worth even trying to save (SPEC B4: "JPEG, PNG, GIF and WebP only").
// It's re-implemented here rather than imported so this package's own
// "pure function, no dependency beyond the standard library on this
// path" stays true — the two are small enough that keeping them
// independent isn't a real cost.
func sniffSupportedImage(data []byte) bool {
	if isWebP(data) {
		return true
	}
	switch detected := http.DetectContentType(data); {
	case strings.HasPrefix(detected, "image/jpeg"),
		strings.HasPrefix(detected, "image/png"),
		strings.HasPrefix(detected, "image/gif"):
		return true
	default:
		return false
	}
}

func isWebP(data []byte) bool {
	return len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP"
}

// isEMF recognises an Enhanced Metafile by its header: record type 1, and the
// " EMF" signature at byte 40.
func isEMF(data []byte) bool {
	return len(data) >= 44 && data[0] == 1 && data[1] == 0 && data[2] == 0 && data[3] == 0 &&
		string(data[40:44]) == " EMF"
}

// isWMF recognises a Windows Metafile: the placeable header's magic number
// 0x9AC6CDD7, or a standard header (type 1 or 2 memory or disk metafile, a
// header size of 9 words, and version 0x0100 or 0x0300).
func isWMF(data []byte) bool {
	if len(data) >= 4 && data[0] == 0xD7 && data[1] == 0xCD && data[2] == 0xC6 && data[3] == 0x9A {
		return true
	}
	if len(data) < 18 {
		return false
	}
	typ := int(data[0]) | int(data[1])<<8
	size := int(data[2]) | int(data[3])<<8
	version := int(data[4]) | int(data[5])<<8
	return (typ == 1 || typ == 2) && size == 9 && (version == 0x0100 || version == 0x0300)
}
