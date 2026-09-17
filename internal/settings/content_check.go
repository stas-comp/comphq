package settings

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/kb"
)

type contentCheckPageData struct {
	Current  string
	Articles []kb.ContentCheckArticle
}

// handleContentCheck lists every article Content check should flag
// (SPEC gates 1.36, 1.49): an image that failed to fetch, or a
// Word-import placeholder still in the body.
func (h *Handlers) handleContentCheck(w http.ResponseWriter, r *http.Request) {
	articles, err := h.articles.ListContentCheck(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := contentCheckPageData{Current: "content-check", Articles: articles}
	h.srv.RenderFrame(w, r, http.StatusOK, "settings-content-check.html", "Content check", data)
}
