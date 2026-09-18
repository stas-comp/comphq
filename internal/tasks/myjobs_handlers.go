package tasks

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

type myJobsPageData struct {
	CurrentView string
	Lane        Lane
	UpForGrabs  []simpleCardView
	Message     string
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
	h.renderMyJobs(w, r, http.StatusOK, person, "")
}

func (h *Handlers) renderMyJobs(w http.ResponseWriter, r *http.Request, status int, person people.Person, message string) {
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

	h.srv.RenderFrame(w, r, status, "tasks-myjobs.html", "Tasks", myJobsPageData{
		CurrentView: "myjobs",
		Lane:        LaneForPerson(teamTasks, person),
		UpForGrabs:  views,
		Message:     message,
	})
}

// conflictMessage renders SPEC gate 2.36's exact wording for the second
// taker: "<Name> has just taken this job." Up for grabs only ever lists
// jobs with zero assignees to begin with, so Names is in practice
// always the single person who won the race.
func conflictMessage(names []string) string {
	return strings.Join(names, ", ") + " has just taken this job."
}

// handleTakeTask serves gates 2.34/2.35's Take, by drag (an optional
// drop position) or the plain Take it button (none), and gate 2.36's
// conflict: if someone beat this request to it, My jobs re-renders
// (status 409, so a plain form submission shows it exactly like any
// other response) with the exact message and no change.
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
	err = h.tasks.Take(r.Context(), id, beforeID, afterID, person.ID, today)
	var taken *TakenError
	switch {
	case err == nil, errors.Is(err, ErrTaskNotFound):
		http.Redirect(w, r, "/tasks", http.StatusFound)
	case errors.As(err, &taken):
		h.renderMyJobs(w, r, http.StatusConflict, person, conflictMessage(taken.Names))
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
