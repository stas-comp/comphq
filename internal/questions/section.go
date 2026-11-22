package questions

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

// Handlers serves the Questions pages.
type Handlers struct {
	srv   *app.Server
	store *Store
}

// Section registers Questions as one more section (SPEC B14.6): one package, one
// registration line in cmd/comphq, its own migrations, templates and static files.
// In the sidebar it sits between Calendar and Settings (gate 8.70).
func Section(srv *app.Server) app.Section {
	h := &Handlers{srv: srv, store: &Store{DB: srv.DB}}
	return app.Section{
		MigrationName: "questions",
		Nav:           &app.NavItem{Label: "Questions", Path: "/questions", Icon: "questions"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /questions", h.handleList)
			mux.HandleFunc("POST /questions", h.handleAsk)
			mux.HandleFunc("GET /questions/version", h.handleVersion)
			mux.HandleFunc("GET /questions/{id}", h.handleQuestion)
			mux.HandleFunc("POST /questions/{id}/answers", h.handleAddAnswer)
			mux.HandleFunc("POST /questions/{id}/answered", h.handleMarkAnswered)
			mux.HandleFunc("POST /questions/{id}/unanswered", h.handleMarkUnanswered)
		},
	}
}

type listPageData struct {
	List    ListResult
	ShowAll bool
	Message string
	// TypedText and TypedDetails keep what was typed when a question is refused.
	TypedText    string
	TypedDetails string
}

type questionPageData struct {
	Question Question
	Message  string
	// TypedAnswer keeps what was typed when an answer is refused.
	TypedAnswer string
}

func (h *Handlers) renderList(w http.ResponseWriter, r *http.Request, status int, message, text, details string) {
	today := app.Today(h.srv.TestMode)
	showAll := r.URL.Query().Get("all") == "1"
	list, err := h.store.List(r.Context(), today, showAll)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, status, "questions-list.html", "Questions", listPageData{
		List: list, ShowAll: showAll, Message: message, TypedText: text, TypedDetails: details,
	})
}

func (h *Handlers) handleList(w http.ResponseWriter, r *http.Request) {
	h.renderList(w, r, http.StatusOK, "", "", "")
}

// handleAsk serves POST /questions (gate 8.71). A refusal brings the page back with
// its plain words and what was typed; nothing is saved.
func (h *Handlers) handleAsk(w http.ResponseWriter, r *http.Request) {
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "not signed in", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	text, details := r.FormValue("text"), r.FormValue("details")
	_, err := h.store.Ask(r.Context(), text, details, person.ID, app.Now(h.srv.TestMode))
	switch {
	case err == nil:
		http.Redirect(w, r, "/questions", http.StatusFound)
	case isRefusal(err):
		h.renderList(w, r, http.StatusOK, err.Error(), text, details)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func isRefusal(err error) bool {
	switch err {
	case ErrEmptyQuestion, ErrQuestionTooLong, ErrDetailsTooLong, ErrEmptyAnswer, ErrAnswerTooLong, ErrQuestionNotFound, ErrAnswerNotFound:
		return true
	}
	return false
}

func (h *Handlers) renderQuestion(w http.ResponseWriter, r *http.Request, id int64, status int, message, typedAnswer string) {
	q, err := h.store.Get(r.Context(), id, app.Today(h.srv.TestMode))
	if err == ErrQuestionNotFound {
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, status, "questions-question.html", "Question", questionPageData{Question: q, Message: message, TypedAnswer: typedAnswer})
}

func (h *Handlers) handleQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.renderQuestion(w, r, id, http.StatusOK, "", "")
}

// question reads what every write on a question needs: its id and who is acting.
func (h *Handlers) question(w http.ResponseWriter, r *http.Request) (int64, people.Person, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, people.Person{}, false
	}
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "not signed in", http.StatusForbidden)
		return 0, people.Person{}, false
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return 0, people.Person{}, false
	}
	return id, person, true
}

// done answers a write: back to the question's page on success, or the page again
// with the plain refusal. A question that is gone shows the same as any missing page.
func (h *Handlers) done(w http.ResponseWriter, r *http.Request, id int64, err error, typedAnswer string) {
	switch {
	case err == nil:
		http.Redirect(w, r, "/questions/"+strconv.FormatInt(id, 10), http.StatusFound)
	case err == ErrQuestionNotFound:
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
	case isRefusal(err):
		h.renderQuestion(w, r, id, http.StatusOK, err.Error(), typedAnswer)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handlers) handleAddAnswer(w http.ResponseWriter, r *http.Request) {
	id, person, ok := h.question(w, r)
	if !ok {
		return
	}
	text := r.FormValue("text")
	_, err := h.store.AddAnswer(r.Context(), id, text, person.ID, app.Now(h.srv.TestMode))
	h.done(w, r, id, err, text)
}

func (h *Handlers) handleMarkAnswered(w http.ResponseWriter, r *http.Request) {
	id, person, ok := h.question(w, r)
	if !ok {
		return
	}
	h.done(w, r, id, h.store.SetAnswered(r.Context(), id, person.ID, app.Now(h.srv.TestMode)), "")
}

func (h *Handlers) handleMarkUnanswered(w http.ResponseWriter, r *http.Request) {
	id, person, ok := h.question(w, r)
	if !ok {
		return
	}
	h.done(w, r, id, h.store.ClearAnswered(r.Context(), id, person.ID, app.Now(h.srv.TestMode)), "")
}

// handleVersion answers with the questions change counter, which refresh.js polls
// (and the Briefing's version adds in, gate 8.79): one cheap query.
func (h *Handlers) handleVersion(w http.ResponseWriter, r *http.Request) {
	v, err := h.store.Version(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"version": v})
}
