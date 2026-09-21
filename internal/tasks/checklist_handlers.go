package tasks

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

// Steps inside a job (SPEC B10). Six POST routes, each an ordinary form
// that answers a redirect back to the task page; a form carrying
// fragment=1 (the task window) is answered with the steps list itself
// instead. Every one works with no script at all (gate 5.08).

// Notice codes; the wording for each lives in steps.html (SPEC B8: user-
// facing words stay in templates).
const (
	noticeEmpty   = "empty"
	noticeTooLong = "too-long"
	noticeTooMany = "too-many"
	noticeRemoved = "removed"
	noticeGone    = "gone"
)

// stepsState is what the steps list needs to know beyond the database: a
// notice to show, the words a refused form had typed (so nothing is lost),
// which step is being renamed, and which removal offers an Undo.
type stepsState struct {
	Notice     string
	AddText    string
	RenameID   int64
	RenameText string
	UndoID     int64
}

// stepView is one row of the list.
type stepView struct {
	ID         int64
	Text       string
	Done       bool
	DoneBy     string
	DoneDate   string
	Renaming   bool
	RenameText string
	MoveUp     app.IconButton
	MoveDown   app.IconButton
	Remove     app.IconButton
}

// stepsView is the whole Steps section, drawn by the one partial that both
// the task window and the task page include (SPEC B10.3).
type stepsView struct {
	TaskID    int64
	Steps     []stepView
	Total     int
	DoneCount int
	AllDone   bool
	// Notice is a message about what was just tried (too many steps, an
	// empty word, a step somebody else removed), shown at the top of the
	// list with AddText still in the box.
	Notice  string
	AddText string
	// UndoID and UndoText are the step just removed, offered back.
	UndoID   int64
	UndoText string
	InWindow bool
	// Version is the change counter as it was when this list was read, so a
	// script can tell later that somebody else has changed something.
	Version int64
	// The two limits, so the wording of a refusal can name them.
	MaxSteps int
	MaxChars int
}

// stateFromQuery reads the two things a plain page can be asked to show:
// one step in its rename form, and the Undo for one just-removed step.
func stateFromQuery(r *http.Request) stepsState {
	var st stepsState
	if id, err := strconv.ParseInt(r.URL.Query().Get("rename_step"), 10, 64); err == nil {
		st.RenameID = id
	}
	if id, err := strconv.ParseInt(r.URL.Query().Get("undo_step"), 10, 64); err == nil {
		st.UndoID = id
	}
	return st
}

// buildStepsView reads a job's steps and lays them out with st applied.
func (h *Handlers) buildStepsView(r *http.Request, taskID int64, today time.Time, inWindow bool, st stepsState) (stepsView, error) {
	// The counter is read first: a change landing between the two reads then
	// shows up as a newer counter next time, never as a missed change.
	version, err := h.tasks.Version(r.Context())
	if err != nil {
		return stepsView{}, err
	}
	all, err := h.tasks.ListStepsWithRemoved(r.Context(), taskID, today)
	if err != nil {
		return stepsView{}, err
	}
	v := stepsView{Version: version, TaskID: taskID, InWindow: inWindow, Notice: st.Notice, AddText: st.AddText, MaxSteps: MaxSteps, MaxChars: MaxStepChars}

	var live []Step
	for _, s := range all {
		switch {
		case !s.Removed:
			live = append(live, s)
		case s.ID == st.UndoID:
			v.UndoID, v.UndoText = s.ID, s.Text
		}
	}
	v.Total = len(live)
	for i, s := range live {
		if s.Done {
			v.DoneCount++
		}
		row := stepView{
			ID: s.ID, Text: s.Text, Done: s.Done, DoneBy: s.DoneByName, DoneDate: s.DoneDate,
			MoveUp:   app.NewIconButton(app.IconMoveUp, s.Text, i == 0),
			MoveDown: app.NewIconButton(app.IconMoveDown, s.Text, i == len(live)-1),
			Remove:   app.NewIconButton(app.IconRemove, s.Text, false),
		}
		if s.ID == st.RenameID {
			row.Renaming = true
			row.RenameText = s.Text
			if st.RenameText != "" {
				row.RenameText = st.RenameText
			}
		}
		v.Steps = append(v.Steps, row)
	}
	v.AllDone = v.Total > 0 && v.DoneCount == v.Total
	return v, nil
}

// stepAction is what every step route has in common: who, which job, and
// (for the five routes that name one) which step.
type stepAction struct {
	taskID int64
	stepID int64
	person people.Person
}

// parseStepAction reads the path and the form. wantStep is false only for
// adding, which has no step yet. It writes the refusal itself and returns
// false when the request can't be acted on.
func (h *Handlers) parseStepAction(w http.ResponseWriter, r *http.Request, wantStep bool) (stepAction, bool) {
	var a stepAction
	var err error
	if a.taskID, err = strconv.ParseInt(r.PathValue("id"), 10, 64); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return a, false
	}
	if wantStep {
		if a.stepID, err = strconv.ParseInt(r.PathValue("stepID"), 10, 64); err != nil {
			http.Error(w, "invalid step id", http.StatusBadRequest)
			return a, false
		}
	}
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "not signed in", http.StatusForbidden)
		return a, false
	}
	a.person = person
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return a, false
	}
	return a, true
}

// noticeFor turns a store refusal into the notice code the page words, and
// says whether the refusal is one a person can cause (and so is shown) as
// opposed to a fault.
func noticeFor(err error) (string, bool) {
	switch err {
	case ErrEmptyStep:
		return noticeEmpty, true
	case ErrStepTooLong:
		return noticeTooLong, true
	case ErrTooManySteps:
		return noticeTooMany, true
	case ErrStepRemoved:
		return noticeRemoved, true
	case ErrStepNotFound:
		return noticeGone, true
	}
	return "", false
}

// stepDone ends a step route. Success is a redirect back to the task page
// (or, for the window, the steps list as a fragment). A refusal a person
// can cause (a limit, a step somebody else removed) is shown in place with
// what they typed kept, and changes nothing; anything else is a fault.
func (h *Handlers) stepDone(w http.ResponseWriter, r *http.Request, a stepAction, err error, st stepsState, redirectQuery, anchor string) {
	if err == ErrTaskNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		notice, ok := noticeFor(err)
		if !ok {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		st.Notice = notice
	} else {
		// What a refused form had typed is only kept while it was refused.
		st.AddText, st.RenameID, st.RenameText = "", 0, ""
	}

	if wantsFragment(r) {
		status := http.StatusOK
		if err != nil {
			status = http.StatusUnprocessableEntity
		}
		h.renderSteps(w, r, a.taskID, status, st)
		return
	}
	if err == nil {
		http.Redirect(w, r, "/tasks/"+strconv.FormatInt(a.taskID, 10)+redirectQuery+anchor, http.StatusFound)
		return
	}
	h.renderDetails(w, r, a.taskID, http.StatusOK, "", nil, nil, &st)
}

// handleStepsList is GET /tasks/{id}/steps: the list on its own, for the
// script to swap in (a rename form, a cancel, another computer's change).
// Asked for as a page it just goes to the task page.
func (h *Handlers) handleStepsList(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if _, err := h.tasks.Get(r.Context(), id, app.Today(h.srv.TestMode)); err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !wantsFragment(r) {
		http.Redirect(w, r, "/tasks/"+strconv.FormatInt(id, 10)+"#steps", http.StatusFound)
		return
	}
	h.renderSteps(w, r, id, http.StatusOK, stateFromQuery(r))
}

// renderSteps answers a fragment request with just the Steps section.
func (h *Handlers) renderSteps(w http.ResponseWriter, r *http.Request, taskID int64, status int, st stepsState) {
	view, err := h.buildStepsView(r, taskID, app.Today(h.srv.TestMode), true, st)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderPartial(w, status, "tasks-steps", view)
}

func (h *Handlers) handleAddStep(w http.ResponseWriter, r *http.Request) {
	a, ok := h.parseStepAction(w, r, false)
	if !ok {
		return
	}
	text := r.FormValue("text")
	_, err := h.tasks.AddStep(r.Context(), a.taskID, text, a.person.ID, app.Today(h.srv.TestMode))
	h.stepDone(w, r, a, err, stepsState{AddText: text}, "", "#steps")
}

func (h *Handlers) handleTickStep(w http.ResponseWriter, r *http.Request) {
	a, ok := h.parseStepAction(w, r, true)
	if !ok {
		return
	}
	// The form carries the state it asks for. A checkbox-style form sends
	// the ticked value only when ticked, so "no 1 among the values" is an
	// untick; a form that sent no value at all isn't a tick form.
	values := r.PostForm["done"]
	if len(values) == 0 {
		http.Error(w, "missing done", http.StatusBadRequest)
		return
	}
	done := false
	for _, v := range values {
		if v == "1" {
			done = true
		}
	}
	err := h.tasks.SetStepDone(r.Context(), a.taskID, a.stepID, done, a.person.ID, app.Today(h.srv.TestMode))
	h.stepDone(w, r, a, err, stepsState{}, "", "#step-"+strconv.FormatInt(a.stepID, 10))
}

func (h *Handlers) handleRenameStep(w http.ResponseWriter, r *http.Request) {
	a, ok := h.parseStepAction(w, r, true)
	if !ok {
		return
	}
	text := r.FormValue("text")
	err := h.tasks.RenameStep(r.Context(), a.taskID, a.stepID, text, a.person.ID, app.Today(h.srv.TestMode))
	// A refused rename comes back still in its rename form, with the typed
	// words in the box — unless the step is gone, when there is no form.
	st := stepsState{RenameID: a.stepID, RenameText: text}
	if err == ErrStepRemoved || err == ErrStepNotFound {
		st.RenameID, st.RenameText = 0, ""
	}
	h.stepDone(w, r, a, err, st, "", "#step-"+strconv.FormatInt(a.stepID, 10))
}

func (h *Handlers) handleRemoveStep(w http.ResponseWriter, r *http.Request) {
	a, ok := h.parseStepAction(w, r, true)
	if !ok {
		return
	}
	err := h.tasks.RemoveStep(r.Context(), a.taskID, a.stepID, a.person.ID, app.Today(h.srv.TestMode))
	// On success the Undo is offered straight away: on the redirected page
	// through the query, or in the fragment through the state.
	st := stepsState{}
	query := ""
	if err == nil {
		st.UndoID = a.stepID
		query = "?undo_step=" + strconv.FormatInt(a.stepID, 10)
	}
	h.stepDone(w, r, a, err, st, query, "#steps")
}

func (h *Handlers) handleRestoreStep(w http.ResponseWriter, r *http.Request) {
	a, ok := h.parseStepAction(w, r, true)
	if !ok {
		return
	}
	err := h.tasks.RestoreStep(r.Context(), a.taskID, a.stepID, a.person.ID, app.Today(h.srv.TestMode))
	h.stepDone(w, r, a, err, stepsState{}, "", "#step-"+strconv.FormatInt(a.stepID, 10))
}

// handleMoveStep moves a step one place ("direction" up or down) or to an
// absolute 1-based place ("position", which is what dragging sends).
func (h *Handlers) handleMoveStep(w http.ResponseWriter, r *http.Request) {
	a, ok := h.parseStepAction(w, r, true)
	if !ok {
		return
	}
	today := app.Today(h.srv.TestMode)
	var err error
	if raw := strings.TrimSpace(r.FormValue("position")); raw != "" {
		position, perr := strconv.Atoi(raw)
		if perr != nil {
			http.Error(w, "invalid position", http.StatusBadRequest)
			return
		}
		err = h.tasks.MoveStepTo(r.Context(), a.taskID, a.stepID, position, a.person.ID, today)
	} else {
		err = h.tasks.MoveStep(r.Context(), a.taskID, a.stepID, r.FormValue("direction"), a.person.ID, today)
		if err == ErrInvalidDirection {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	h.stepDone(w, r, a, err, stepsState{}, "", "#step-"+strconv.FormatInt(a.stepID, 10))
}
