package kb

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/stas-comp/comphq/internal/kb/docx"
	kbhtml "github.com/stas-comp/comphq/internal/kb/html"
	"github.com/stas-comp/comphq/internal/people"
)

// docxUnreadableMessage is SPEC gate 1.51's exact wording for a file
// Comp HQ can't read at all (old .doc, password-protected, PDF, or a
// damaged file) — the docx package can't tell these apart, so they all
// share this one message.
const docxUnreadableMessage = "Comp HQ can't open this file. Open it in Word, remove any password, choose File → Save As → Word Document (.docx), then try again."

// handleImportDocx converts an uploaded .docx into editor-ready HTML
// (SPEC B4): it never creates or changes an article — Publish, following
// the normal path, is what does that. Pictures are stored through the
// same image-store function as a paste or drop upload, and the result
// runs through the same sanitiser as publishing.
func (h *Handlers) handleImportDocx(w http.ResponseWriter, r *http.Request) {
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// A generous slack over the 50 MB limit for multipart boundaries and
	// field headers, not the file itself — the actual limit is enforced
	// below, on just the file's own bytes, the same way image uploads are.
	r.Body = http.MaxBytesReader(w, r.Body, docx.MaxUploadSize+64*1024)
	if err := r.ParseMultipartForm(1024 * 1024); err != nil {
		http.Error(w, "That file is larger than 50 MB.", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "internal error", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, docx.MaxUploadSize+1))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	result, err := docx.Convert(bytes.NewReader(data), int64(len(data)), header.Filename)
	switch err {
	case nil:
	case docx.ErrTooLarge:
		http.Error(w, "That file is larger than 50 MB.", http.StatusBadRequest)
		return
	case docx.ErrUnreadable:
		http.Error(w, docxUnreadableMessage, http.StatusBadRequest)
		return
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	bodyHTML := result.HTML
	for _, img := range result.Images {
		stored, err := h.images.Save(r.Context(), bytes.NewReader(img.Bytes), person.ID)
		if err != nil {
			// Leaving the token in place means Sanitize (which drops any
			// img not already under /images/) removes this one picture
			// rather than the whole import failing over it.
			continue
		}
		bodyHTML = strings.ReplaceAll(bodyHTML, img.Token, "/images/"+stored.SHA256+"."+stored.Ext)
	}

	sanitized := kbhtml.Sanitize(bodyHTML)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"title": result.Title,
		"html":  sanitized,
		"notes": result.Notes,
	})
}
