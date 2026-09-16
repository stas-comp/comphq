// Package settings holds People management, About, and the "Set up this
// computer" page (SPEC A9, P1-15).
package settings

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

type Handlers struct {
	srv    *app.Server
	people *people.Store
}

func Section(srv *app.Server) app.Section {
	h := &Handlers{srv: srv, people: srv.PeopleStore()}
	return app.Section{
		Nav: &app.NavItem{Label: "Settings", Path: "/settings", Icon: "/static/theme/icons/settings.svg"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /settings", h.handleIndex)
			mux.HandleFunc("GET /settings/people", h.handlePeople)
			mux.HandleFunc("POST /settings/people/rename", h.handleRename)
			mux.HandleFunc("POST /settings/people/remove", h.handleRemove)
			mux.HandleFunc("GET /settings/about", h.handleAbout)
			mux.HandleFunc("GET /settings/setup", h.handleSetup)
		},
	}
}

func (h *Handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/settings/people", http.StatusFound)
}
