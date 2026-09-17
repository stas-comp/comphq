package kb

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/people"
)

// articleFormData backs the shared new/edit editor template (SPEC gates
// 1.14, 1.15, 1.20). VersionNo is carried as a hidden field for P1-25's
// edit-conflict detection; nothing reads it back yet.
type articleFormData struct {
	ID         int64
	VersionNo  int
	Title      string
	BodyHTML   string
	BodyHTMLJS template.JS // BodyHTML, JSON-encoded, for the editor's initial content
	CategoryID int64
	Categories []Category
	Message    string
}

func newArticleFormData(id int64, versionNo int, title, bodyHTML string, categoryID int64, categories []Category, message string) (articleFormData, error) {
	encoded, err := json.Marshal(bodyHTML)
	if err != nil {
		return articleFormData{}, err
	}
	return articleFormData{
		ID:         id,
		VersionNo:  versionNo,
		Title:      title,
		BodyHTML:   bodyHTML,
		BodyHTMLJS: template.JS(encoded),
		CategoryID: categoryID,
		Categories: categories,
		Message:    message,
	}, nil
}

func (h *Handlers) handleNewArticle(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categories.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	data, err := newArticleFormData(0, 0, "", "", 0, categories, "")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-editor.html", "New article", data)
}

// articlePageData renders one article. ImageFailures is set only right
// after a publish that couldn't copy every external image (SPEC gate
// 1.19: "the editor response lists the failures") — a plain view of an
// already-published article never has any.
type articlePageData struct {
	Article       Article
	BodyHTML      template.HTML
	Category      Category
	ImageFailures []string
}

func (h *Handlers) renderArticlePage(w http.ResponseWriter, r *http.Request, status int, article Article, imageFailures []string) {
	category, err := h.categories.Get(article.CategoryID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// article.BodyHTML was sanitised on save (SPEC B4); it's the one place
	// article content bypasses html/template's auto-escaping.
	h.srv.RenderFrame(w, r, status, "kb-article.html", article.Title, articlePageData{
		Article:       article,
		BodyHTML:      template.HTML(article.BodyHTML),
		Category:      category,
		ImageFailures: imageFailures,
	})
}

func (h *Handlers) handleViewArticle(w http.ResponseWriter, r *http.Request) {
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
	h.renderArticlePage(w, r, http.StatusOK, article, nil)
}

func (h *Handlers) handleEditArticle(w http.ResponseWriter, r *http.Request) {
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
	categories, err := h.categories.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	data, err := newArticleFormData(article.ID, article.VersionNo, article.Title, article.BodyHTML, article.CategoryID, categories, "")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-editor.html", "Edit "+article.Title, data)
}

// handleSaveArticle handles both a new article and an edit: the form
// carries a hidden id field, empty for a new article (SPEC gate 1.14 and
// 1.22: every publish, new or edited, records a version).
func (h *Handlers) handleSaveArticle(w http.ResponseWriter, r *http.Request) {
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var id int64
	if idStr := r.FormValue("id"); idStr != "" {
		var err error
		id, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
	}
	categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid category_id", http.StatusBadRequest)
		return
	}

	article, err := h.articles.Publish(r.Context(), ArticleInput{
		ID:         id,
		CategoryID: categoryID,
		Title:      r.FormValue("title"),
		BodyHTML:   r.FormValue("body_html"),
	}, person.ID)
	if err == ErrArticleEmptyTitle {
		categories, listErr := h.categories.List()
		if listErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		data, dataErr := newArticleFormData(id, 0, r.FormValue("title"), r.FormValue("body_html"), categoryID, categories, "Please type a title.")
		if dataErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		h.srv.RenderFrame(w, r, http.StatusOK, "kb-editor.html", "New article", data)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// The text is still saved even when a picture couldn't be copied
	// (SPEC gate 1.19), so this renders the article directly (status 200)
	// with the failures listed, rather than a clean redirect to its URL.
	if len(article.ImageFailures) > 0 {
		h.renderArticlePage(w, r, http.StatusOK, article, article.ImageFailures)
		return
	}
	http.Redirect(w, r, "/kb/articles/"+strconv.FormatInt(article.ID, 10), http.StatusFound)
}
