// Package kb holds the Knowledge Base: categories, articles, versions,
// search and images (SPEC A5).
package kb

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
)

type Handlers struct {
	srv        *app.Server
	categories *CategoryStore
}

func Section(srv *app.Server) app.Section {
	h := &Handlers{srv: srv, categories: &CategoryStore{DB: srv.DB}}
	return app.Section{
		MigrationName: "kb", // migrations/kb/0001_search.sql (P1-08), 0002_categories.sql (P1-17)
		Nav:           &app.NavItem{Label: "Knowledge Base", Path: "/kb", Icon: "/static/theme/icons/kb.svg"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /kb", h.handleHome)
			mux.HandleFunc("GET /kb/categories", h.handleCategories)
			mux.HandleFunc("POST /kb/categories", h.handleCreateCategory)
			mux.HandleFunc("POST /kb/categories/{id}/rename", h.handleRenameCategory)
			mux.HandleFunc("POST /kb/categories/{id}/move-up", h.handleMoveCategoryUp)
			mux.HandleFunc("POST /kb/categories/{id}/move-down", h.handleMoveCategoryDown)
			mux.HandleFunc("POST /kb/categories/{id}/delete", h.handleDeleteCategory)
		},
	}
}
