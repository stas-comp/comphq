package tasks

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

type Handlers struct {
	srv    *app.Server
	tasks  *Store
	people *people.Store
}

func Section(srv *app.Server) app.Section {
	h := &Handlers{
		srv:    srv,
		tasks:  &Store{DB: srv.DB},
		people: srv.PeopleStore(),
	}
	return app.Section{
		MigrationName: "tasks",
		Nav:           &app.NavItem{Label: "Tasks", Path: "/tasks", Icon: "/static/theme/icons/tasks.svg"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /tasks", h.handleRedirectToBoard) // My jobs lands here from P2-10
			mux.HandleFunc("GET /tasks/board", h.handleBoard)
			mux.HandleFunc("POST /tasks", h.handleCreateTask)
		},
	}
}
