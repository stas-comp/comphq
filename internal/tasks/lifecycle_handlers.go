package tasks

import (
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

// simpleCardView is one row of Finished tasks or Removed tasks (SPEC
// gates 2.08/2.09) — a plain summary, not a board card: neither list is
// drag-reorderable, so it needs none of cardView's move-button state.
type simpleCardView struct {
	ID        int64
	Title     string
	SizeLabel string
	DueDate   string
	Assignees []assigneeView
}

func newSimpleCardView(t Task, meID int64) simpleCardView {
	views := make([]assigneeView, 0, len(t.Assignees))
	for _, a := range t.Assignees {
		views = append(views, assigneeView{
			Name:       a.Name,
			Initials:   InitialsFor(a.Name),
			ColorClass: AvatarClass(a.PersonID, meID),
			Removed:    a.Removed,
		})
	}
	return simpleCardView{ID: t.ID, Title: t.Title, SizeLabel: sizeLabels[t.Size], DueDate: t.DueDate, Assignees: views}
}

type finishedPageData struct {
	Tasks []simpleCardView
}

// handleFinished lists Done tasks that have aged off the board (SPEC
// gate 2.08), each with a Reopen action.
func (h *Handlers) handleFinished(w http.ResponseWriter, r *http.Request) {
	today := app.Today(h.srv.TestMode)
	tasks, err := h.tasks.ListFinished(r.Context(), today)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	views := make([]simpleCardView, 0, len(tasks))
	for _, t := range tasks {
		views = append(views, newSimpleCardView(t, currentPersonID(r)))
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "tasks-finished.html", "Finished tasks", finishedPageData{Tasks: views})
}

// handleReopenTask serves gate 2.08's Reopen action. A missing task
// (the list changed under us — someone else reopened or removed it
// first) just re-renders the current list, the same "board changed
// under us" pattern handleMoveTask already uses.
func (h *Handlers) handleReopenTask(w http.ResponseWriter, r *http.Request) {
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

	today := app.Today(h.srv.TestMode)
	switch err := h.tasks.Reopen(r.Context(), id, person.ID, today); err {
	case nil, ErrTaskNotFound:
		http.Redirect(w, r, "/tasks/finished", http.StatusFound)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

type removedPageData struct {
	Tasks []simpleCardView
}

// handleRemoved lists removed tasks (SPEC gate 2.09), each with a
// Restore action. Nothing here can ever be permanently deleted.
func (h *Handlers) handleRemoved(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.tasks.ListRemoved(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	views := make([]simpleCardView, 0, len(tasks))
	for _, t := range tasks {
		views = append(views, newSimpleCardView(t, currentPersonID(r)))
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "tasks-removed.html", "Removed tasks", removedPageData{Tasks: views})
}

// handleRemoveTask serves gate 2.09's "Remove task" action, from the
// task details page. The task disappears from the board immediately,
// which is the visible confirmation the action took effect.
func (h *Handlers) handleRemoveTask(w http.ResponseWriter, r *http.Request) {
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

	today := app.Today(h.srv.TestMode)
	switch err := h.tasks.Remove(r.Context(), id, person.ID, today); err {
	case nil:
		http.Redirect(w, r, "/tasks/board", http.StatusFound)
	case ErrTaskNotFound:
		http.NotFound(w, r)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// handleRestoreTask serves gate 2.09's Restore action, from the Removed
// tasks list.
func (h *Handlers) handleRestoreTask(w http.ResponseWriter, r *http.Request) {
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

	today := app.Today(h.srv.TestMode)
	switch err := h.tasks.Restore(r.Context(), id, person.ID, today); err {
	case nil, ErrTaskNotFound:
		http.Redirect(w, r, "/tasks/removed", http.StatusFound)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
