package tasks

import (
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

type teamPageData struct {
	CurrentView string
	Lanes       []Lane
}

// handleTeam serves the Team view (SPEC gates 2.24, 2.25, 2.27, 2.30):
// an Unassigned lane, then one lane per active person, each showing
// Working on now, numbered Up next, and — for a person, not Unassigned
// — a workload row.
func (h *Handlers) handleTeam(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.tasks.ListForTeamView(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	activePeople, err := h.people.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.srv.RenderFrame(w, r, http.StatusOK, "tasks-team.html", "Team", teamPageData{
		CurrentView: "team",
		Lanes:       BuildLanes(tasks, activePeople),
	})
}

// handleAssignTask serves gate 2.28's drag and "Assign to…" button
// (SPEC B4's POST /tasks/{id}/assign): from_person/to_person are each
// optional, but at least one is required.
func (h *Handlers) handleAssignTask(w http.ResponseWriter, r *http.Request) {
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

	var fromPersonID, toPersonID int64
	if v := r.FormValue("from_person"); v != "" {
		if fromPersonID, err = strconv.ParseInt(v, 10, 64); err != nil {
			http.Error(w, "invalid from_person", http.StatusBadRequest)
			return
		}
	}
	if v := r.FormValue("to_person"); v != "" {
		if toPersonID, err = strconv.ParseInt(v, 10, 64); err != nil {
			http.Error(w, "invalid to_person", http.StatusBadRequest)
			return
		}
	}

	// Give back (My jobs) reuses this same endpoint (SPEC B4), so a
	// plain (non-JS) form submission needs to land back on whichever
	// page it came from — see redirectTargetFrom.
	redirectTarget := redirectTargetFrom(r)
	if redirectTarget == "" {
		redirectTarget = "/tasks/team"
	}

	today := app.Today(h.srv.TestMode)
	switch err := h.tasks.Assign(r.Context(), id, fromPersonID, toPersonID, person.ID, today); err {
	case nil, ErrTaskNotFound:
		http.Redirect(w, r, redirectTarget, http.StatusFound)
	case ErrInvalidAssign:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
