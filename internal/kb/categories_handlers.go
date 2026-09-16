package kb

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
)

type categoriesPageData struct {
	Categories []Category
	Message    string
}

func (h *Handlers) handleCategories(w http.ResponseWriter, r *http.Request) {
	h.renderCategories(w, r, http.StatusOK, "")
}

func (h *Handlers) renderCategories(w http.ResponseWriter, r *http.Request, status int, message string) {
	categories, err := h.categories.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, status, "kb-categories.html", "Categories", categoriesPageData{
		Categories: categories,
		Message:    message,
	})
}

// categoryPageData is the SPEC gate 1.14 destination: a category's own
// page, listing its published articles.
type categoryPageData struct {
	Category Category
	Articles []CategoryArticle
}

func (h *Handlers) handleViewCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	category, err := h.categories.Get(id)
	if err == sql.ErrNoRows {
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	articles, err := h.articles.ListByCategory(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-category.html", category.Name, categoryPageData{
		Category: category,
		Articles: articles,
	})
}

func (h *Handlers) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	switch _, err := h.categories.Create(r.FormValue("name")); err {
	case nil:
		http.Redirect(w, r, "/kb/categories", http.StatusFound)
	case ErrEmptyName:
		h.renderCategories(w, r, http.StatusOK, "Please type a name.")
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handlers) handleRenameCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	switch err := h.categories.Rename(id, r.FormValue("name")); err {
	case nil:
		http.Redirect(w, r, "/kb/categories", http.StatusFound)
	case ErrEmptyName:
		h.renderCategories(w, r, http.StatusOK, "Please type a name.")
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handlers) handleMoveCategoryUp(w http.ResponseWriter, r *http.Request) {
	h.move(w, r, h.categories.MoveUp)
}

func (h *Handlers) handleMoveCategoryDown(w http.ResponseWriter, r *http.Request) {
	h.move(w, r, h.categories.MoveDown)
}

func (h *Handlers) move(w http.ResponseWriter, r *http.Request, fn func(int64) error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := fn(id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/kb/categories", http.StatusFound)
}

// handleDeleteCategory refuses with a plain explanation when the category
// still has articles (SPEC gate 1.12).
func (h *Handlers) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	count, err := h.categories.Delete(id)
	switch err {
	case nil:
		http.Redirect(w, r, "/kb/categories", http.StatusFound)
	case ErrCategoryHasArticles:
		word := "articles"
		if count == 1 {
			word = "article"
		}
		h.renderCategories(w, r, http.StatusOK,
			fmt.Sprintf("This category still has %d %s. Move or archive them first.", count, word))
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
