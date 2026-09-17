package kb

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/stas-comp/comphq/internal/kb/images"
	"github.com/stas-comp/comphq/internal/people"
)

// handleUploadImage stores a pasted or dropped image (SPEC gates 1.16,
// 1.17). The request body is the raw file bytes, not a multipart form —
// editor.js posts the File object directly.
func (h *Handlers) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	img, err := h.images.Save(r.Context(), r.Body, person.ID)
	switch err {
	case nil:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"src": "/images/" + img.SHA256 + "." + img.Ext})
	case images.ErrTooLarge:
		// Save only reads up to its own limit, so bytes beyond that are
		// still arriving on the wire. Drain them before responding —
		// otherwise the client's fetch() sees a broken connection instead
		// of this response (observed as "That file could not be
		// uploaded." from editor.js's own catch, not this message).
		io.Copy(io.Discard, r.Body)
		http.Error(w, "That file is larger than 20 MB.", http.StatusBadRequest)
	case images.ErrUnsupportedType:
		http.Error(w, "That file isn't a JPEG, PNG, GIF or WebP image.", http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// handleServeImage serves a stored image (SPEC B4): Content-Type from the
// stored mime type (never trusting the request's own extension), nosniff,
// and immutable caching — filenames are content-hashed, so a given URL's
// bytes never change.
func (h *Handlers) handleServeImage(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	ext := strings.TrimPrefix(path.Ext(filename), ".")
	sha := strings.TrimSuffix(filename, "."+ext)

	img, err := h.images.Get(sha, ext)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", img.MIME)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, h.images.Path(sha, ext))
}
