package tasks

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

// handleAddLink serves POST /tasks/{id}/links (gate 7.40, B13.6): other_id and
// kind (first, then or related). A refusal (gate 7.45) shows its plain message
// on the job's page and changes nothing.
func (h *Handlers) handleAddLink(w http.ResponseWriter, r *http.Request) {
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
	otherID, err := strconv.ParseInt(r.FormValue("other_id"), 10, 64)
	if err != nil {
		h.renderDetails(w, r, id, http.StatusOK, ErrLinkOtherRemoved.Error(), nil, nil, nil)
		return
	}

	err = h.tasks.AddLink(r.Context(), id, otherID, r.FormValue("kind"), person.ID, app.Today(h.srv.TestMode))
	h.afterLinkChange(w, r, id, err)
}

// handleRemoveLink serves POST /tasks/{id}/links/{linkID}/remove (gate 7.44).
func (h *Handlers) handleRemoveLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	linkID, err := strconv.ParseInt(r.PathValue("linkID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid link id", http.StatusBadRequest)
		return
	}
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "not signed in", http.StatusForbidden)
		return
	}
	err = h.tasks.RemoveLink(r.Context(), id, linkID, person.ID, app.Today(h.srv.TestMode))
	h.afterLinkChange(w, r, id, err)
}

// afterLinkChange answers a link action. From a page: back to the job's page on
// success, or the page again with the refusal's own words. From the window or
// the script (fragment=1): the Linked jobs section as it now is, with the
// refusal's words in it, so the screen changes without a reload.
func (h *Handlers) afterLinkChange(w http.ResponseWriter, r *http.Request, id int64, err error) {
	var loop *LinkLoopError
	refused := errors.As(err, &loop) || errors.Is(err, ErrLinkSelf) || errors.Is(err, ErrLinkExists) || errors.Is(err, ErrLinkTooMany) ||
		errors.Is(err, ErrLinkOtherRemoved) || errors.Is(err, ErrLinkKind) || errors.Is(err, ErrLinkNotFound)
	switch {
	case errors.Is(err, ErrTaskNotFound):
		http.NotFound(w, r)
	case err != nil && !refused:
		http.Error(w, "internal error", http.StatusInternalServerError)
	case wantsFragment(r):
		notice := ""
		if err != nil {
			notice = err.Error()
		}
		data, derr := h.buildLinksData(r.Context(), id, "", true, notice, app.Today(h.srv.TestMode))
		if derr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		h.srv.RenderPartial(w, http.StatusOK, "tasks-links", data)
	case err == nil:
		http.Redirect(w, r, "/tasks/"+strconv.FormatInt(id, 10), http.StatusFound)
	default:
		h.renderDetails(w, r, id, http.StatusOK, err.Error(), nil, nil, nil)
	}
}
