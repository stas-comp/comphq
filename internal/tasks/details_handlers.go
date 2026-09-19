package tasks

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/app/format"
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
	ID    int64
	Title string
	Notes string
	Size  string
	Stage string
	// DueDate is the field's value, day first; DueLabel is the date as a
	// sentence reads it ("Sat 19 Sep 2026"), for reading.
	DueDate string
	// InWindow is true when this is drawn in the task window (a fragment).
	InWindow   bool
	SizeLabel  string
	StageLabel string
	DueLabel   string
	Assignees  []assigneeView
	Stages     []stageOption
	People     []personOption
	Activity   []Activity
	Message    string
}

// handleTaskDetails serves a task's own page, unchanged (SPEC gate 4.27:
// Calendar links point at it and a refreshed window must land somewhere
// real). With ?fragment=1 it is the task window's face for the same task:
// reading by default, or the edit form with &mode=edit (gates 4.23, 4.24).
func (h *Handlers) handleTaskDetails(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.renderDetails(w, r, id, http.StatusOK, "", nil, nil)
}

// renderDetails draws the task page, or — for the window — its reading or
// editing fragment. typed and typedPeople, when given, are what a refused
// save had entered, so the form comes back with it instead of the stored
// values.
func (h *Handlers) renderDetails(w http.ResponseWriter, r *http.Request, id int64, status int, message string, typed *UpdateInput, typedPeople []int64) {
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

	me := currentPersonID(r)
	views := make([]assigneeView, 0, len(task.Assignees))
	for _, a := range task.Assignees {
		views = append(views, assigneeView{Name: a.Name, Initials: InitialsFor(a.Name), ColorClass: AvatarClass(a.PersonID, me), Removed: a.Removed})
	}
	dueLabel := ""
	if due, err := time.Parse("2006-01-02", task.DueDate); err == nil {
		dueLabel = format.Date(due)
	}

	data := detailsPageData{
		ID: task.ID, Title: task.Title, Notes: task.Notes, Size: task.Size, Stage: task.Stage, DueDate: format.DayFirst(task.DueDate),
		InWindow:  wantsFragment(r),
		SizeLabel: sizeLabels[task.Size], StageLabel: stageLabels[task.Stage], DueLabel: dueLabel,
		Assignees: views,
		Stages:    allStages,
		People:    buildPersonOptions(activePeople, task.Assignees),
		Activity:  activity,
		Message:   message,
	}
	if typed != nil {
		data.Title, data.Notes, data.Size, data.Stage, data.DueDate = typed.Title, typed.Notes, typed.Size, typed.Stage, format.DayFirst(typed.DueDate)
		chosen := make(map[int64]bool, len(typedPeople))
		for _, id := range typedPeople {
			chosen[id] = true
		}
		for i := range data.People {
			data.People[i].Selected = chosen[data.People[i].ID]
		}
	}

	if data.InWindow {
		// A save is always answered with the form (a refused one), and a plain
		// GET with &mode=edit is the form; anything else is the reading view.
		name := "tasks-window-read"
		if r.Method == http.MethodPost || r.FormValue("mode") == "edit" {
			name = "tasks-window-edit"
		}
		h.srv.RenderPartial(w, status, name, data)
		return
	}
	h.srv.RenderFrame(w, r, status, "tasks-details.html", "Task: "+task.Title, data)
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
		DueDate:   format.NormaliseDate(r.FormValue("due_date")),
		PersonIDs: personIDs,
	}

	// From the task window the form posts as a fragment: a saved edit answers
	// 204, a refused one 422 with the form again (typed values kept).
	refusedStatus := http.StatusOK
	if wantsFragment(r) {
		refusedStatus = http.StatusUnprocessableEntity
	}
	today := app.Today(h.srv.TestMode)
	switch err := h.tasks.Update(r.Context(), id, input, person.ID, today); err {
	case nil:
		if wantsFragment(r) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, "/tasks/"+strconv.FormatInt(id, 10), http.StatusFound)
	case ErrEmptyTitle:
		h.renderDetails(w, r, id, refusedStatus, "Please type a title.", &input, personIDs)
	case ErrInvalidDate:
		h.renderDetails(w, r, id, refusedStatus, invalidDateMessage, &input, personIDs)
	case ErrTaskNotFound:
		http.NotFound(w, r)
	case ErrInvalidSize, ErrInvalidStage:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
