package tasks

import (
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

type myJobsPageData struct {
	CurrentView string
	Lane        Lane
	UpForGrabs  []simpleCardView
}

// handleMyJobs serves the Tasks home screen (SPEC gates 2.32, 2.33): the
// signed-in person's own lane, built by the same code as their Team
// lane, beside Up for grabs — jobs and ideas nobody is on.
func (h *Handlers) handleMyJobs(w http.ResponseWriter, r *http.Request) {
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "not signed in", http.StatusForbidden)
		return
	}

	teamTasks, err := h.tasks.ListForTeamView(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	grabs, err := h.tasks.ListUpForGrabs(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	views := make([]simpleCardView, 0, len(grabs))
	for _, t := range grabs {
		views = append(views, newSimpleCardView(t))
	}

	h.srv.RenderFrame(w, r, http.StatusOK, "tasks-myjobs.html", "Tasks", myJobsPageData{
		CurrentView: "myjobs",
		Lane:        LaneForPerson(teamTasks, person),
		UpForGrabs:  views,
	})
}

// handleTakeTask serves gates 2.34/2.35's Take, by drag (an optional
// drop position) or the plain Take it button (none). SPEC B4's 409-
// with-names conflict response is P2-11's job — for now an already-
// taken job just falls back to re-showing the current state, the same
// "changed under us" pattern handleMoveTask already uses for a task
// gone from the list.
func (h *Handlers) handleTakeTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "not signed in", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	var beforeID, afterID int64
	if v := r.FormValue("before_id"); v != "" {
		if beforeID, err = strconv.ParseInt(v, 10, 64); err != nil {
			http.Error(w, "invalid before_id", http.StatusBadRequest)
			return
		}
	}
	if v := r.FormValue("after_id"); v != "" {
		if afterID, err = strconv.ParseInt(v, 10, 64); err != nil {
			http.Error(w, "invalid after_id", http.StatusBadRequest)
			return
		}
	}

	today := app.Today(h.srv.TestMode)
	switch err := h.tasks.Take(r.Context(), id, beforeID, afterID, person.ID, today); err {
	case nil, ErrTaskNotFound, ErrTaskAlreadyTaken:
		http.Redirect(w, r, "/tasks", http.StatusFound)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
