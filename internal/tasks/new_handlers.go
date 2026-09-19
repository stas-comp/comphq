package tasks

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app/format"
)

// sizeOption is one entry in a size drop-down.
type sizeOption struct {
	Value string
	Label string
}

var sizeOptions = []sizeOption{{"S", "Small"}, {"M", "Medium"}, {"L", "Large"}}

// newTaskPageData is the new-task page (SPEC gates 4.22, 4.28): the same
// fields the task window opens with, as an ordinary page. + Add task is a
// link to it, so a task can always be added even when the window can't open.
type newTaskPageData struct {
	Message string
	// InWindow is true when the form is drawn inside the task window (a
	// fragment): Cancel closes the window, and the form posts as a fragment.
	InWindow bool
	Title    string
	Notes    string
	Size    string
	Stage   string
	DueDate string // as typed, day first
	Sizes   []sizeOption
	Stages  []stageOption
	People  []personOption
}

// handleNewTaskForm serves GET /tasks/new: an empty form, In Ideas, Medium.
// With ?fragment=1 it is just the form, for the task window.
func (h *Handlers) handleNewTaskForm(w http.ResponseWriter, r *http.Request) {
	h.renderNewTask(w, r, http.StatusOK, "", CreateInput{Size: "M", Stage: StageIdea}, nil)
}

// renderNewTask draws the page with whatever was typed, so a refused save
// (no title, a date that isn't a day) keeps everything the person entered.
func (h *Handlers) renderNewTask(w http.ResponseWriter, r *http.Request, status int, message string, input CreateInput, selected []int64) {
	activePeople, err := h.people.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	chosen := make(map[int64]bool, len(selected))
	for _, id := range selected {
		chosen[id] = true
	}
	options := buildPersonOptions(activePeople, nil)
	for i := range options {
		options[i].Selected = chosen[options[i].ID]
	}

	stages := make([]stageOption, 0, len(Stages))
	for _, stage := range Stages {
		stages = append(stages, stageOption{Stage: stage, Label: stageLabels[stage]})
	}
	size, stage := input.Size, input.Stage
	if size == "" {
		size = "M"
	}
	if stage == "" {
		stage = StageIdea
	}

	data := newTaskPageData{
		Message:  message,
		InWindow: wantsFragment(r),
		Title:    input.Title,
		Notes:    input.Notes,
		Size:     size,
		Stage:    stage,
		DueDate:  format.DayFirst(input.DueDate),
		Sizes:    sizeOptions,
		Stages:   stages,
		People:   options,
	}
	if data.InWindow {
		h.srv.RenderPartial(w, status, "tasks-new-form", data)
		return
	}
	h.srv.RenderFrame(w, r, status, "tasks-new.html", "Add task", data)
}
