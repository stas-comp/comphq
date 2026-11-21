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

// SectionOption changes how Section is wired.
type SectionOption func(*Handlers)

// WithArticleFinder lets jobs link Knowledge Base articles (D-105). cmd/comphq
// passes the Knowledge Base's implementation, so Tasks never imports it.
func WithArticleFinder(f ArticleFinder) SectionOption {
	return func(h *Handlers) { h.tasks.Articles = f }
}

func Section(srv *app.Server, opts ...SectionOption) app.Section {
	h := &Handlers{
		srv:    srv,
		tasks:  &Store{DB: srv.DB},
		people: srv.PeopleStore(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return app.Section{
		MigrationName: "tasks",
		Nav: &app.NavItem{Label: "Tasks", Path: "/tasks", Icon: "tasks", Search: &app.SearchSetting{
			Kind: "jobs", Action: "/tasks/search", JSONURL: "/tasks/search.json", Label: "Search jobs",
			InsteadLabel: "Search articles instead", InsteadHref: "/kb/search",
		}},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /tasks", h.withRepeats(h.handleMyJobs))
			mux.HandleFunc("GET /tasks/board", h.withRepeats(h.handleBoard))
			mux.HandleFunc("GET /tasks/new", h.handleNewTaskForm)
			mux.HandleFunc("POST /tasks", h.handleCreateTask)
			mux.HandleFunc("GET /tasks/version", h.withRepeats(h.handleVersion))
			mux.HandleFunc("GET /tasks/team", h.withRepeats(h.handleTeam))
			mux.HandleFunc("GET /tasks/finished", h.handleFinished)
			mux.HandleFunc("GET /tasks/removed", h.handleRemoved)
			mux.HandleFunc("GET /tasks/search", h.handleSearchPage)
			mux.HandleFunc("GET /tasks/search.json", h.handleSearchJSON)
			mux.HandleFunc("GET /tasks/articles.json", h.handleArticlesJSON)
			mux.HandleFunc("GET /tasks/{id}", h.withRepeats(h.handleTaskDetails))
			mux.HandleFunc("GET /tasks/{id}/people", h.handleTaskPeople)
			mux.HandleFunc("POST /tasks/{id}", h.handleUpdateTask)
			mux.HandleFunc("POST /tasks/{id}/move", h.handleMoveTask)
			mux.HandleFunc("POST /tasks/{id}/assign", h.handleAssignTask)
			mux.HandleFunc("POST /tasks/{id}/take", h.handleTakeTask)
			mux.HandleFunc("POST /tasks/{id}/reopen", h.handleReopenTask)
			mux.HandleFunc("POST /tasks/{id}/remove", h.handleRemoveTask)
			mux.HandleFunc("POST /tasks/{id}/links", h.handleAddLink)
			mux.HandleFunc("POST /tasks/{id}/links/{linkID}/remove", h.handleRemoveLink)
			mux.HandleFunc("POST /tasks/{id}/articles", h.handleAddArticle)
			mux.HandleFunc("POST /tasks/{id}/articles/{linkID}/remove", h.handleRemoveArticle)
			mux.HandleFunc("POST /tasks/{id}/places", h.handleAddPlace)
			mux.HandleFunc("POST /tasks/{id}/places/{placeID}/remove", h.handleRemovePlace)
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

// withRepeats runs the Saturday reset (weekly.go) and the yearly pass (yearly.go)
// before a request that shows jobs, so a NAS that was off on a Saturday, or
// through a yearly job's show day, catches up the first time anyone opens Comp HQ
// (gates 7.35, 8.47). With nothing to do each is one cheap query. A failure is
// logged and the page is shown anyway: the jobs are still right, the pass will
// simply be tried again on the next request.
func (h *Handlers) withRepeats(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := h.tasks.ResetWeekly(r.Context(), app.Today(h.srv.TestMode)); err != nil {
			log.Printf("tasks: weekly reset: %v", err)
		}
		if _, err := h.tasks.ResetYearly(r.Context(), app.Today(h.srv.TestMode)); err != nil {
			log.Printf("tasks: yearly pass: %v", err)
		}
		next(w, r)
	}
}
