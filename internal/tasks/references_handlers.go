package tasks

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

// Endpoints for articles and COSMOS places on a job (SPEC B14.2, D-105). All
// writes are POSTs under the origin check; every refusal is the store's own
// plain message, and changes nothing.

type pickedArticle struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
}

// handleArticlesJSON serves GET /tasks/articles.json?q=&task=: the "Link an
// article" picker (gate 8.11), up to ten published articles matching q, leaving
// out those already linked to task.
func (h *Handlers) handleArticlesJSON(w http.ResponseWriter, r *http.Request) {
	taskID, _ := strconv.ParseInt(r.URL.Query().Get("task"), 10, 64)
	found, err := h.tasks.SearchArticles(r.Context(), r.URL.Query().Get("q"), taskID)
	if err != nil {
		log.Printf("tasks: article picker: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	out := make([]pickedArticle, len(found))
	for i, a := range found {
		out[i] = pickedArticle{ID: a.ID, Title: a.Title, Category: a.Category}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"results": out})
}

// referenceRequest reads what every reference endpoint needs: the job and who
// is acting.
func referenceRequest(w http.ResponseWriter, r *http.Request) (id int64, person people.Person, ok bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, people.Person{}, false
	}
	person, found := people.FromContext(r.Context())
	if !found {
		http.Error(w, "not signed in", http.StatusForbidden)
		return 0, people.Person{}, false
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return 0, people.Person{}, false
	}
	return id, person, true
}

func (h *Handlers) handleAddArticle(w http.ResponseWriter, r *http.Request) {
	id, person, ok := referenceRequest(w, r)
	if !ok {
		return
	}
	articleID, err := strconv.ParseInt(r.FormValue("article_id"), 10, 64)
	if err != nil {
		h.afterReferenceChange(w, r, id, ErrArticleGone)
		return
	}
	h.afterReferenceChange(w, r, id, h.tasks.AddArticle(r.Context(), id, articleID, person.ID, app.Now(h.srv.TestMode)))
}

func (h *Handlers) handleRemoveArticle(w http.ResponseWriter, r *http.Request) {
	id, person, ok := referenceRequest(w, r)
	if !ok {
		return
	}
	linkID, err := strconv.ParseInt(r.PathValue("linkID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid link id", http.StatusBadRequest)
		return
	}
	h.afterReferenceChange(w, r, id, h.tasks.RemoveArticle(r.Context(), id, linkID, person.ID, app.Now(h.srv.TestMode)))
}

func (h *Handlers) handleAddPlace(w http.ResponseWriter, r *http.Request) {
	id, person, ok := referenceRequest(w, r)
	if !ok {
		return
	}
	h.afterReferenceChange(w, r, id, h.tasks.AddPlace(r.Context(), id, r.FormValue("name"), r.FormValue("path"), person.ID, app.Now(h.srv.TestMode)))
}

func (h *Handlers) handleRemovePlace(w http.ResponseWriter, r *http.Request) {
	id, person, ok := referenceRequest(w, r)
	if !ok {
		return
	}
	placeID, err := strconv.ParseInt(r.PathValue("placeID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid place id", http.StatusBadRequest)
		return
	}
	h.afterReferenceChange(w, r, id, h.tasks.RemovePlace(r.Context(), id, placeID, person.ID, app.Now(h.srv.TestMode)))
}

// isReferenceRefusal says whether err is one of the plain refusals (as opposed
// to a fault).
func isReferenceRefusal(err error) bool {
	for _, refusal := range []error{
		ErrPlacePath, ErrPlaceNameLong, ErrPlaceDuplicate, ErrTooManyPlaces, ErrArticleLinked,
		ErrTooManyArticles, ErrArticleGone, ErrArticleArchived, ErrJobRemoved, ErrReferenceGone,
	} {
		if errors.Is(err, refusal) {
			return true
		}
	}
	return false
}

// afterReferenceChange answers a reference action: back to the job's page on
// success, or the page again with the refusal's own words.
func (h *Handlers) afterReferenceChange(w http.ResponseWriter, r *http.Request, id int64, err error) {
	switch {
	case errors.Is(err, ErrTaskNotFound):
		http.NotFound(w, r)
	case err != nil && !isReferenceRefusal(err):
		log.Printf("tasks: references: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	case err == nil:
		http.Redirect(w, r, "/tasks/"+strconv.FormatInt(id, 10), http.StatusFound)
	default:
		h.renderDetails(w, r, id, http.StatusOK, err.Error(), nil, nil, nil)
	}
}
