package tasks

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/app"
)

// personChoice is one line of a job's people list: a current person, marked
// when they're already on the job.
type personChoice struct {
	ID   int64
	Name string
	On   bool
}

type taskPeopleData struct {
	ID            int64
	Title         string
	PeopleChoices []personChoice
}

// handleTaskPeople serves a job's people list (SPEC gate 4.20): the
// short menu behind a card's people circles. As a page it is where the
// circles' link goes when script isn't running; with ?fragment=1 it is
// just the menu, for board.js to show on the card. Either way each name is
// a button that posts to the existing assign endpoint.
func (h *Handlers) handleTaskPeople(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	task, err := h.tasks.Get(r.Context(), id, app.Today(h.srv.TestMode))
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	everyone, err := h.people.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	onJob := make(map[int64]bool, len(task.Assignees))
	for _, a := range task.Assignees {
		onJob[a.PersonID] = true
	}
	choices := make([]personChoice, 0, len(everyone))
	for _, p := range everyone {
		choices = append(choices, personChoice{ID: p.ID, Name: p.Name, On: onJob[p.ID]})
	}
	data := taskPeopleData{ID: task.ID, Title: task.Title, PeopleChoices: choices}

	if r.URL.Query().Get("fragment") == "1" {
		h.srv.RenderPartial(w, http.StatusOK, "tasks-people-panel", data)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "tasks-people.html", "People on "+task.Title, data)
}
