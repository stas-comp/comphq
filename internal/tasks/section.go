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
		Nav:           &app.NavItem{Label: "Tasks", Path: "/tasks", Icon: "tasks"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /tasks", h.handleMyJobs)
			mux.HandleFunc("GET /tasks/board", h.handleBoard)
			mux.HandleFunc("GET /tasks/new", h.handleNewTaskForm)
			mux.HandleFunc("POST /tasks", h.handleCreateTask)
			mux.HandleFunc("GET /tasks/version", h.handleVersion)
			mux.HandleFunc("GET /tasks/team", h.handleTeam)
			mux.HandleFunc("GET /tasks/finished", h.handleFinished)
			mux.HandleFunc("GET /tasks/removed", h.handleRemoved)
			mux.HandleFunc("GET /tasks/{id}", h.handleTaskDetails)
			mux.HandleFunc("GET /tasks/{id}/people", h.handleTaskPeople)
			mux.HandleFunc("POST /tasks/{id}", h.handleUpdateTask)
			mux.HandleFunc("POST /tasks/{id}/move", h.handleMoveTask)
			mux.HandleFunc("POST /tasks/{id}/assign", h.handleAssignTask)
			mux.HandleFunc("POST /tasks/{id}/take", h.handleTakeTask)
			mux.HandleFunc("POST /tasks/{id}/reopen", h.handleReopenTask)
			mux.HandleFunc("POST /tasks/{id}/remove", h.handleRemoveTask)
			mux.HandleFunc("POST /tasks/{id}/restore", h.handleRestoreTask)
			mux.HandleFunc("POST /tasks/{id}/steps", h.handleAddStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}", h.handleRenameStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}/tick", h.handleTickStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}/remove", h.handleRemoveStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}/restore", h.handleRestoreStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}/move", h.handleMoveStep)
		},
	}
}
