// Package settings holds People management, About, and the "Set up this
// computer" page (SPEC A9, P1-15).
package settings

import (
	"context"
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/kb"
	"github.com/stas-comp/comphq/internal/kb/images"
	"github.com/stas-comp/comphq/internal/people"
)

// CSVSource is the one method Settings needs from Tasks and Calendar to
// build tasks.csv, events.csv and steps.csv (SPEC B2: settings depends on the other
// sections' export interfaces, not their internals). Both stores satisfy
// it directly, since it deals only in plain text rows.
type CSVSource interface {
	ExportRows(ctx context.Context) ([][]string, error)
}

type Handlers struct {
	tasks    CSVSource
	events   CSVSource
	steps    CSVSource
	srv      *app.Server
	people   *people.Store
	articles *kb.ArticleStore
	images   *images.Store
}

func Section(srv *app.Server, tasksCSV, eventsCSV, stepsCSV CSVSource) app.Section {
	imagesStore := &images.Store{DB: srv.DB, DataDir: srv.DataDir}
	h := &Handlers{
		srv:      srv,
		tasks:    tasksCSV,
		events:   eventsCSV,
		steps:    stepsCSV,
		people:   srv.PeopleStore(),
		articles: &kb.ArticleStore{DB: srv.DB, Images: imagesStore},
		images:   imagesStore,
	}
	return app.Section{
		Nav: &app.NavItem{Label: "Settings", Path: "/settings", Icon: "settings"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /settings", h.handleIndex)
			mux.HandleFunc("GET /settings/people", h.handlePeople)
			mux.HandleFunc("POST /settings/people/rename", h.handleRename)
			mux.HandleFunc("POST /settings/people/remove", h.handleRemove)
			mux.HandleFunc("GET /settings/export", h.handleExport)
			mux.HandleFunc("GET /settings/backups", h.handleBackups)
			mux.HandleFunc("GET /settings/content-check", h.handleContentCheck)
			mux.HandleFunc("GET /settings/about", h.handleAbout)
			mux.HandleFunc("GET /settings/setup", h.handleSetup)
		},
	}
}

func (h *Handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/settings/people", http.StatusFound)
}
