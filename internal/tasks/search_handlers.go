package tasks

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/app"
)

// searchJobs runs the query for the request: q, and an optional exclude id
// (the Linked jobs picker leaves the job itself out, B13.6).
func (h *Handlers) searchJobs(r *http.Request) ([]JobHit, error) {
	exclude, _ := strconv.ParseInt(r.URL.Query().Get("exclude"), 10, 64)
	return h.tasks.SearchJobs(r.Context(), r.URL.Query().Get("q"), exclude, app.Today(h.srv.TestMode))
}

// handleSearchJSON serves the top bar's live panel on Tasks pages (gates
// 7.20, 7.21): up to 20 jobs for the words typed.
func (h *Handlers) handleSearchJSON(w http.ResponseWriter, r *http.Request) {
	hits, err := h.searchJobs(r)
	if err != nil {
		log.Printf("tasks: job search: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"results": hits})
}

type searchPageData struct {
	Query   string
	Results []JobHit
	// WindowClose is the task window's close icon: a result opens the job in
	// the window where there is one (gate 7.23).
	WindowClose app.IconButton
}

// handleSearchPage is the full results page Enter goes to (gate 7.23), with
// the same results as the panel. A result is a link to the job's own page.
func (h *Handlers) handleSearchPage(w http.ResponseWriter, r *http.Request) {
	hits, err := h.searchJobs(r)
	if err != nil {
		log.Printf("tasks: job search: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "tasks-search.html", "Search jobs", searchPageData{
		Query:       r.URL.Query().Get("q"),
		Results:     hits,
		WindowClose: app.NewIconButton(app.IconClose, "", false),
	})
}
