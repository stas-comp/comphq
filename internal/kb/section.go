// Package kb holds the Knowledge Base: categories, articles, versions,
// search and images (SPEC A5).
package kb

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/kb/images"
)

type Handlers struct {
	srv        *app.Server
	categories *CategoryStore
	articles   *ArticleStore
	images     *images.Store
}

func Section(srv *app.Server) app.Section {
	imagesStore := &images.Store{DB: srv.DB, DataDir: srv.DataDir}
	h := &Handlers{
		srv:        srv,
		categories: &CategoryStore{DB: srv.DB},
		articles:   &ArticleStore{DB: srv.DB, Images: imagesStore},
		images:     imagesStore,
	}
	return app.Section{
		MigrationName: "kb", // migrations/kb/0001_search.sql (P1-08), 0002_categories.sql (P1-17), 0003_article_versions.sql (P1-19), 0004_images.sql (P1-21)
		Nav:           &app.NavItem{Label: "Knowledge Base", Path: "/kb", Icon: "/static/theme/icons/kb.svg"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /kb", h.handleHome)
			mux.HandleFunc("GET /kb/categories", h.handleCategories)
			mux.HandleFunc("POST /kb/categories", h.handleCreateCategory)
			mux.HandleFunc("GET /kb/categories/{id}", h.handleViewCategory)
			mux.HandleFunc("POST /kb/categories/{id}/rename", h.handleRenameCategory)
			mux.HandleFunc("POST /kb/categories/{id}/move-up", h.handleMoveCategoryUp)
			mux.HandleFunc("POST /kb/categories/{id}/move-down", h.handleMoveCategoryDown)
			mux.HandleFunc("POST /kb/categories/{id}/delete", h.handleDeleteCategory)
			mux.HandleFunc("GET /kb/new", h.handleNewArticle)
			mux.HandleFunc("GET /kb/articles/{id}", h.handleViewArticle)
			mux.HandleFunc("GET /kb/articles/{id}/edit", h.handleEditArticle)
			mux.HandleFunc("POST /kb/articles", h.handleSaveArticle)
			mux.HandleFunc("GET /kb/articles/{id}/history", h.handleArticleHistory)
			mux.HandleFunc("GET /kb/articles/{id}/versions/{n}", h.handleViewVersion)
			mux.HandleFunc("POST /kb/articles/{id}/versions/{n}/restore", h.handleRestoreVersion)
			mux.HandleFunc("POST /kb/articles/{id}/archive", h.handleArchiveArticle)
			mux.HandleFunc("POST /kb/articles/{id}/unarchive", h.handleUnarchiveArticle)
			mux.HandleFunc("GET /kb/archived", h.handleArchivedList)
			mux.HandleFunc("GET /kb/search", h.handleSearchResultsPage)
			mux.HandleFunc("GET /kb/search.json", h.handleSearchJSON)
			mux.HandleFunc("POST /kb/images", h.handleUploadImage)
			mux.HandleFunc("GET /images/{filename}", h.handleServeImage)
		},
	}
}
