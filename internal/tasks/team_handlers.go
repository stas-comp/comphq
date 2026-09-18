package tasks

import "net/http"

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
