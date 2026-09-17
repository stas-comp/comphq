package kb

import (
	"context"
	"database/sql"
	"html/template"
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/people"
)

type historyPageData struct {
	Article  Article
	Versions []ArticleVersion
}

// handleArticleHistory lists every version, newest first (SPEC gate 1.22).
func (h *Handlers) handleArticleHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	article, err := h.articles.Get(id)
	if err == sql.ErrNoRows {
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	versions, err := h.articles.History(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-history.html", "History: "+article.Title, historyPageData{
		Article: article, Versions: versions,
	})
}

type versionPageData struct {
	Article  Article
	Version  ArticleVersion
	BodyHTML template.HTML
}

// handleViewVersion shows one old version exactly as it was (SPEC gate
// 1.23).
func (h *Handlers) handleViewVersion(w http.ResponseWriter, r *http.Request) {
	id, n, err := parseArticleAndVersion(r)
	if err != nil {
		http.Error(w, "invalid id or version", http.StatusBadRequest)
		return
	}
	article, err := h.articles.Get(id)
	if err == sql.ErrNoRows {
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	version, err := h.articles.Version(id, n)
	if err == sql.ErrNoRows {
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-version.html", article.Title, versionPageData{
		Article:  article,
		Version:  version,
		BodyHTML: template.HTML(version.BodyHTML),
	})
}

// handleRestoreVersion makes an old version current again (SPEC gate
// 1.23).
func (h *Handlers) handleRestoreVersion(w http.ResponseWriter, r *http.Request) {
	id, n, err := parseArticleAndVersion(r)
	if err != nil {
		http.Error(w, "invalid id or version", http.StatusBadRequest)
		return
	}
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err := h.articles.Restore(r.Context(), id, n, person.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/kb/articles/"+strconv.FormatInt(id, 10), http.StatusFound)
}

func parseArticleAndVersion(r *http.Request) (id int64, n int, err error) {
	id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	n, err = strconv.Atoi(r.PathValue("n"))
	return id, n, err
}

// handleArchiveArticle removes an article from its category and search
// (SPEC gate 1.24).
func (h *Handlers) handleArchiveArticle(w http.ResponseWriter, r *http.Request) {
	h.setArticleStatus(w, r, h.articles.Archive)
}

// handleUnarchiveArticle puts an article back in its category and search
// (SPEC gate 1.24).
func (h *Handlers) handleUnarchiveArticle(w http.ResponseWriter, r *http.Request) {
	h.setArticleStatus(w, r, h.articles.Unarchive)
}

func (h *Handlers) setArticleStatus(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, articleID, personID int64) error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := fn(r.Context(), id, person.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/kb/articles/"+strconv.FormatInt(id, 10), http.StatusFound)
}

type archivedPageData struct {
	Articles []ArchivedArticle
}

// handleArchivedList lists every archived article (SPEC gate 1.24).
func (h *Handlers) handleArchivedList(w http.ResponseWriter, r *http.Request) {
	articles, err := h.articles.ListArchived()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-archived.html", "Archived articles", archivedPageData{Articles: articles})
}
