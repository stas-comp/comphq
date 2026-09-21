package app

import (
	"io/fs"
	"mime"
	"net/http"

	"github.com/stas-comp/comphq"
)

func init() {
	// The container is a bare image with no /etc/mime.types, and Go's own
	// table knows neither of these, so without this the manifest and the
	// icon would be served as whatever the first bytes look like. Browsers
	// read a manifest as JSON either way, but a wrong type is the first thing
	// an install check complains about.
	mime.AddExtensionType(".webmanifest", "application/manifest+json")
	mime.AddExtensionType(".ico", "image/x-icon")
}

// faviconPath is the one icon file, inside the app, that /favicon.ico serves.
const faviconPath = "web/static/theme/icons/comphq.ico"

// handleFavicon answers GET /favicon.ico with the multi-size Comp HQ icon
// (SPEC B11.2, gate 5.22). Every page links its icons explicitly, but some
// browsers, and Windows itself when it draws a shortcut, ask for this address
// and nothing else, so it has to be answered — with the icon, not the
// "Page not found" frame, and without first asking who is using the computer.
func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(comphq.Static, faviconPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(data)
}
