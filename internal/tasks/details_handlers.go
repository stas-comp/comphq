package tasks

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

// personOption is one row of the details page's people multi-select.
type personOption struct {
	ID       int64
	Name     string
	Selected bool
}

// buildPersonOptions lists every active person, plus — SPEC gate
// 2.11 — any already-assigned person who's since been removed: they no
// longer appear when assigning someone new, but a removed assignee must
// stay marked and selected here, or simply re-saving the form without
// touching people would silently unassign them (they'd never have been
// an option to keep selected otherwise).
func buildPersonOptions(activePeople []people.Person, assignees []Assignee) []personOption {
	stillAssigned := make(map[int64]bool, len(assignees))
	for _, a := range assignees {
		stillAssigned[a.PersonID] = true
	}

	options := make([]personOption, 0, len(activePeople))
	for _, p := range activePeople {
		options = append(options, personOption{ID: p.ID, Name: p.Name, Selected: stillAssigned[p.ID]})
		delete(stillAssigned, p.ID)
	}
	for _, a := range assignees {
		if stillAssigned[a.PersonID] {
			options = append(options, personOption{ID: a.PersonID, Name: a.Name + " (removed)", Selected: true})
		}
	}
	return options
}

type detailsPageData struct {
	ID       int64
	Title    string
	Notes    string
	Size     string
	Stage    string
	DueDate  string
	Stages   []stageOption
	People   []personOption
	Activity []Activity
	Message  string
}

func (h *Handlers) handleTaskDetails(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.renderDetails(w, r, id, http.StatusOK, "")
}

func (h *Handlers) renderDetails(w http.ResponseWriter, r *http.Request, id int64, status int, message string) {
	today := app.Today(h.srv.TestMode)
	task, err := h.tasks.Get(r.Context(), id, today)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	activity, err := h.tasks.ListActivity(r.Context(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	activePeople, err := h.people.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	allStages := make([]stageOption, 0, len(Stages))
	for _, stage := range Stages {
		allStages = append(allStages, stageOption{Stage: stage, Label: stageLabels[stage]})
	}

	h.srv.RenderFrame(w, r, status, "tasks-details.html", "Task: "+task.Title, detailsPageData{
		ID: task.ID, Title: task.Title, Notes: task.Notes, Size: task.Size, Stage: task.Stage, DueDate: task.DueDate,
		Stages:   allStages,
		People:   buildPersonOptions(activePeople, task.Assignees),
		Activity: activity,
		Message:  message,
	})
}

func (h *Handlers) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
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
	personIDs, err := parsePersonIDs(r)
	if err != nil {
		http.Error(w, "invalid person_id", http.StatusBadRequest)
		return
	}

	input := UpdateInput{
		Title:     r.FormValue("title"),
		Notes:     r.FormValue("notes"),
		Size:      r.FormValue("size"),
		Stage:     r.FormValue("stage"),
		DueDate:   strings.TrimSpace(r.FormValue("due_date")),
		PersonIDs: personIDs,
	}

	today := app.Today(h.srv.TestMode)
	switch err := h.tasks.Update(r.Context(), id, input, person.ID, today); err {
	case nil:
		http.Redirect(w, r, "/tasks/"+strconv.FormatInt(id, 10), http.StatusFound)
	case ErrEmptyTitle:
		h.renderDetails(w, r, id, http.StatusOK, "Please type a title.")
	case ErrTaskNotFound:
		http.NotFound(w, r)
	case ErrInvalidSize, ErrInvalidStage:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
