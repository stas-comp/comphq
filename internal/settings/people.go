package settings

import (
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/people"
)

type peoplePageData struct {
	Current string
	People  []people.Person
	Message string
}

func (h *Handlers) handlePeople(w http.ResponseWriter, r *http.Request) {
	h.renderPeople(w, r, http.StatusOK, "")
}

func (h *Handlers) renderPeople(w http.ResponseWriter, r *http.Request, status int, message string) {
	list, err := h.people.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	data := peoplePageData{Current: "people", People: list, Message: message}
	h.srv.RenderFrame(w, r, status, "settings-people.html", "People", data)
}

// handleRename lets a staff member correct a name; it shows everywhere
// that person appears, including old history, since history always
// resolves through people.id (SPEC gate 1.05).
func (h *Handlers) handleRename(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.FormValue("person_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid person_id", http.StatusBadRequest)
		return
	}

	switch err := h.people.Rename(id, r.FormValue("name")); err {
	case nil:
		http.Redirect(w, r, "/settings/people", http.StatusFound)
	case people.ErrEmptyName:
		h.renderPeople(w, r, http.StatusOK, "Please type a name.")
	case people.ErrDuplicateName:
		h.renderPeople(w, r, http.StatusOK, "That name is already taken.")
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// handleRemove deactivates a person: they leave the picker and assignment
// lists, but old history still shows their name (SPEC gate 1.05).
func (h *Handlers) handleRemove(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.FormValue("person_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid person_id", http.StatusBadRequest)
		return
	}
	if err := h.people.Deactivate(id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings/people", http.StatusFound)
}
