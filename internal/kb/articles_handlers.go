package kb

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/people"
)

// articleFormData backs the shared new/edit article template (SPEC gate
// 1.14). The body field is a plain textarea until the real editor arrives
// in P1-20 (PLAN.md: "a plain contenteditable stub is fine until P1-20").
type articleFormData struct {
	ID         int64
	Title      string
	BodyHTML   string
	CategoryID int64
	Categories []Category
	Message    string
}

func (h *Handlers) handleNewArticle(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categories.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-article-form.html", "New article", articleFormData{
		Categories: categories,
	})
}

type articlePageData struct {
	Article  Article
	BodyHTML template.HTML
	Category Category
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
	category, err := h.categories.Get(article.CategoryID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// article.BodyHTML was sanitised on save (SPEC B4); it's the one place
	// article content bypasses html/template's auto-escaping.
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-article.html", article.Title, articlePageData{
		Article:  article,
		BodyHTML: template.HTML(article.BodyHTML),
		Category: category,
	})
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
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-article-form.html", "Edit "+article.Title, articleFormData{
		ID:         article.ID,
		Title:      article.Title,
		BodyHTML:   article.BodyHTML,
		CategoryID: article.CategoryID,
		Categories: categories,
	})
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
		h.srv.RenderFrame(w, r, http.StatusOK, "kb-article-form.html", "New article", articleFormData{
			ID:         id,
			Title:      r.FormValue("title"),
			BodyHTML:   r.FormValue("body_html"),
			CategoryID: categoryID,
			Categories: categories,
			Message:    "Please type a title.",
		})
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/kb/articles/"+strconv.FormatInt(article.ID, 10), http.StatusFound)
}
