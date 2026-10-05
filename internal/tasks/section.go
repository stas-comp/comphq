package tasks

import (
	"log"
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
		Nav: &app.NavItem{Label: "Tasks", Path: "/tasks", Icon: "tasks", Search: &app.SearchSetting{
			Kind: "jobs", Action: "/tasks/search", JSONURL: "/tasks/search.json", Label: "Search jobs",
			InsteadLabel: "Search articles instead", InsteadHref: "/kb/search",
		}},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /tasks", h.withWeeklyReset(h.handleMyJobs))
			mux.HandleFunc("GET /tasks/board", h.withWeeklyReset(h.handleBoard))
			mux.HandleFunc("GET /tasks/new", h.handleNewTaskForm)
			mux.HandleFunc("POST /tasks", h.handleCreateTask)
			mux.HandleFunc("GET /tasks/version", h.withWeeklyReset(h.handleVersion))
			mux.HandleFunc("GET /tasks/team", h.withWeeklyReset(h.handleTeam))
			mux.HandleFunc("GET /tasks/finished", h.handleFinished)
			mux.HandleFunc("GET /tasks/removed", h.handleRemoved)
			mux.HandleFunc("GET /tasks/search", h.handleSearchPage)
			mux.HandleFunc("GET /tasks/search.json", h.handleSearchJSON)
			mux.HandleFunc("GET /tasks/{id}", h.withWeeklyReset(h.handleTaskDetails))
			mux.HandleFunc("GET /tasks/{id}/people", h.handleTaskPeople)
			mux.HandleFunc("POST /tasks/{id}", h.handleUpdateTask)
			mux.HandleFunc("POST /tasks/{id}/move", h.handleMoveTask)
			mux.HandleFunc("POST /tasks/{id}/assign", h.handleAssignTask)
			mux.HandleFunc("POST /tasks/{id}/take", h.handleTakeTask)
			mux.HandleFunc("POST /tasks/{id}/reopen", h.handleReopenTask)
			mux.HandleFunc("POST /tasks/{id}/remove", h.handleRemoveTask)
			mux.HandleFunc("POST /tasks/{id}/restore", h.handleRestoreTask)
			mux.HandleFunc("GET /tasks/{id}/steps", h.handleStepsList)
			mux.HandleFunc("POST /tasks/{id}/steps", h.handleAddStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}", h.handleRenameStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}/tick", h.handleTickStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}/remove", h.handleRemoveStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}/restore", h.handleRestoreStep)
			mux.HandleFunc("POST /tasks/{id}/steps/{stepID}/move", h.handleMoveStep)
		},
	}
}

// withWeeklyReset runs the Saturday reset (weekly.go) before a request that
// shows jobs, so a NAS that was off on a Saturday catches up the first time
// anyone opens Comp HQ (gate 7.35). With nothing to do it is one cheap query.
// A failure is logged and the page is shown anyway: the jobs are still right,
// the reset will simply be tried again on the next request.
func (h *Handlers) withWeeklyReset(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := h.tasks.ResetWeekly(r.Context(), app.Today(h.srv.TestMode)); err != nil {
			log.Printf("tasks: weekly reset: %v", err)
		}
		next(w, r)
	}
}
