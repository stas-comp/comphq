package questions

import (
	"encoding/json"
	"html/template"
	"log"
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
		Nav: &app.NavItem{Label: "Questions", Path: "/questions", Icon: "questions", Search: &app.SearchSetting{
			Kind: "questions", Action: "/questions/search", JSONURL: "/questions/search.json", Label: "Search questions",
			Insteads: []app.SearchLink{app.SearchArticlesInstead, app.SearchJobsInstead},
		}},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /questions", h.handleList)
			mux.HandleFunc("POST /questions", h.handleAsk)
			mux.HandleFunc("GET /questions/version", h.handleVersion)
			mux.HandleFunc("GET /questions/removed", h.handleRemoved)
			mux.HandleFunc("GET /questions/search", h.handleSearchPage)
			mux.HandleFunc("GET /questions/search.json", h.handleSearchJSON)
			mux.HandleFunc("GET /questions/{id}", h.handleQuestion)
			mux.HandleFunc("POST /questions/{id}", h.handleEditQuestion)
			mux.HandleFunc("POST /questions/{id}/remove", h.handleRemoveQuestion)
			mux.HandleFunc("POST /questions/{id}/restore", h.handleRestoreQuestion)
			mux.HandleFunc("POST /questions/{id}/answers/{aid}", h.handleEditAnswer)
			mux.HandleFunc("POST /questions/{id}/answers/{aid}/remove", h.handleRemoveAnswer)
			mux.HandleFunc("POST /questions/{id}/answers/{aid}/restore", h.handleRestoreAnswer)
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
	// Answers are the answers with their remove icons; EditingQuestion and
	// EditingAnswer say which edit form is open (?edit=question, ?edit_answer=ID),
	// with what was typed when a refused edit comes back.
	Answers         []answerLine
	EditingQuestion bool
	TypedText       string
	TypedDetails    string
	EditingAnswer   int64
	TypedEdit       string
	// Toast is the Undo message after removing an answer (gate 8.76).
	Toast *toastData
	// RemoveQuestion is the question's remove icon.
	RemoveQuestion app.IconButton
}

// answerLine is one answer with its icon-only remove button (gate 4.11's remove
// icon: removing an answer uses the same icon as everything else).
type answerLine struct {
	Answer
	QuestionID int64
	Remove     app.IconButton
}

type toastData struct {
	QuestionID int64
	AnswerID   int64
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

// edits is what a refused edit had typed, kept in its form.
type edits struct {
	question      bool
	text, details string
	answerID      int64
	answerText    string
}

func (h *Handlers) renderQuestion(w http.ResponseWriter, r *http.Request, id int64, status int, message, typedAnswer string) {
	h.renderQuestionWith(w, r, id, status, message, typedAnswer, edits{})
}

func (h *Handlers) renderQuestionWith(w http.ResponseWriter, r *http.Request, id int64, status int, message, typedAnswer string, e edits) {
	q, err := h.store.Get(r.Context(), id, app.Today(h.srv.TestMode))
	if err == ErrQuestionNotFound {
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	data := questionPageData{Question: q, Message: message, TypedAnswer: typedAnswer, RemoveQuestion: app.NewIconButton(app.IconRemove, "question", false)}
	for _, a := range q.Answers {
		data.Answers = append(data.Answers, answerLine{Answer: a, QuestionID: q.ID, Remove: app.NewIconButton(app.IconRemove, "answer by "+a.By, false)})
	}
	// An edit form opens from ?edit=question or ?edit_answer=ID (plain links, so no
	// script is needed), or stays open with what was typed after a refusal.
	switch {
	case e.question || r.URL.Query().Get("edit") == "question":
		data.EditingQuestion = true
		data.TypedText, data.TypedDetails = q.Text, q.Details
		if e.question {
			data.TypedText, data.TypedDetails = e.text, e.details
		}
	case e.answerID != 0 || r.URL.Query().Get("edit_answer") != "":
		aid := e.answerID
		if aid == 0 {
			aid, _ = strconv.ParseInt(r.URL.Query().Get("edit_answer"), 10, 64)
		}
		for _, a := range q.Answers {
			if a.ID == aid {
				data.EditingAnswer, data.TypedEdit = aid, a.Text
				if e.answerID != 0 {
					data.TypedEdit = e.answerText
				}
			}
		}
	}
	// The Undo message after removing an answer (gate 8.76, as gate 7.15): shown
	// only while that answer really is removed.
	if aid, err := strconv.ParseInt(r.URL.Query().Get("removed_answer"), 10, 64); err == nil {
		if ok, _ := h.store.AnswerRemoved(r.Context(), id, aid); ok {
			data.Toast = &toastData{QuestionID: id, AnswerID: aid}
		}
	}
	h.srv.RenderFrame(w, r, status, "questions-question.html", "Question", data)
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

// ---- editing, removing, restoring (gates 8.75, 8.76)

func (h *Handlers) handleEditQuestion(w http.ResponseWriter, r *http.Request) {
	id, person, ok := h.question(w, r)
	if !ok {
		return
	}
	text, details := r.FormValue("text"), r.FormValue("details")
	err := h.store.EditQuestion(r.Context(), id, text, details, person.ID, app.Now(h.srv.TestMode))
	switch {
	case err == nil:
		http.Redirect(w, r, "/questions/"+strconv.FormatInt(id, 10), http.StatusFound)
	case err == ErrQuestionNotFound:
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
	case isRefusal(err):
		h.renderQuestionWith(w, r, id, http.StatusOK, err.Error(), "", edits{question: true, text: text, details: details})
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handlers) answerID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	aid, err := strconv.ParseInt(r.PathValue("aid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid answer id", http.StatusBadRequest)
		return 0, false
	}
	return aid, true
}

func (h *Handlers) handleEditAnswer(w http.ResponseWriter, r *http.Request) {
	id, person, ok := h.question(w, r)
	if !ok {
		return
	}
	aid, ok := h.answerID(w, r)
	if !ok {
		return
	}
	text := r.FormValue("text")
	err := h.store.EditAnswer(r.Context(), id, aid, text, person.ID, app.Now(h.srv.TestMode))
	switch {
	case err == nil:
		http.Redirect(w, r, "/questions/"+strconv.FormatInt(id, 10), http.StatusFound)
	case err == ErrQuestionNotFound:
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
	case isRefusal(err):
		h.renderQuestionWith(w, r, id, http.StatusOK, err.Error(), "", edits{answerID: aid, answerText: text})
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// handleRemoveAnswer takes an answer off and comes back with the Undo message for
// 10 seconds (gate 8.76, as gate 7.15), an ordinary form that works with no script.
func (h *Handlers) handleRemoveAnswer(w http.ResponseWriter, r *http.Request) {
	id, person, ok := h.question(w, r)
	if !ok {
		return
	}
	aid, ok := h.answerID(w, r)
	if !ok {
		return
	}
	_, err := h.store.RemoveAnswer(r.Context(), id, aid, person.ID, app.Now(h.srv.TestMode))
	switch {
	case err == nil:
		http.Redirect(w, r, "/questions/"+strconv.FormatInt(id, 10)+"?removed_answer="+strconv.FormatInt(aid, 10), http.StatusFound)
	case err == ErrAnswerNotFound || err == ErrQuestionNotFound:
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handlers) handleRestoreAnswer(w http.ResponseWriter, r *http.Request) {
	id, person, ok := h.question(w, r)
	if !ok {
		return
	}
	aid, ok := h.answerID(w, r)
	if !ok {
		return
	}
	err := h.store.RestoreAnswer(r.Context(), id, aid, person.ID, app.Now(h.srv.TestMode))
	if err == ErrAnswerNotFound {
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
		return
	}
	h.done(w, r, id, err, "")
}

func (h *Handlers) handleRemoveQuestion(w http.ResponseWriter, r *http.Request) {
	id, person, ok := h.question(w, r)
	if !ok {
		return
	}
	switch err := h.store.RemoveQuestion(r.Context(), id, person.ID, app.Now(h.srv.TestMode)); err {
	case nil:
		http.Redirect(w, r, "/questions", http.StatusFound)
	case ErrQuestionNotFound:
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handlers) handleRestoreQuestion(w http.ResponseWriter, r *http.Request) {
	id, person, ok := h.question(w, r)
	if !ok {
		return
	}
	switch err := h.store.RestoreQuestion(r.Context(), id, person.ID, app.Now(h.srv.TestMode)); err {
	case nil:
		http.Redirect(w, r, "/questions/"+strconv.FormatInt(id, 10), http.StatusFound)
	case ErrQuestionNotFound:
		h.srv.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

type removedPageData struct {
	Questions []RemovedRow
}

func (h *Handlers) handleRemoved(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.ListRemoved(r.Context(), app.Today(h.srv.TestMode))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "questions-removed.html", "Removed questions", removedPageData{Questions: rows})
}

// ---- search (gate 8.78)

type searchJSONHit struct {
	ID       int64         `json:"id"`
	URL      string        `json:"url"`
	Title    template.HTML `json:"title"`
	Passage  template.HTML `json:"passage"`
	AskedBy  string        `json:"asked_by"`
	Asked    string        `json:"asked"`
	Answers  int           `json:"answers"`
	Answered bool          `json:"answered"`
}

func (h *Handlers) search(r *http.Request) ([]Hit, error) {
	return h.store.Search(r.Context(), r.URL.Query().Get("q"), app.Today(h.srv.TestMode))
}

// handleSearchJSON serves the top bar's live panel on Questions pages.
func (h *Handlers) handleSearchJSON(w http.ResponseWriter, r *http.Request) {
	hits, err := h.search(r)
	if err != nil {
		log.Printf("questions: search: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	out := make([]searchJSONHit, len(hits))
	for i, hit := range hits {
		out[i] = searchJSONHit{ID: hit.ID, URL: hit.URL, Title: hit.Title, Passage: hit.Passage, AskedBy: hit.AskedBy, Asked: hit.AskedLabel, Answers: hit.AnswerCount, Answered: hit.Answered}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"results": out})
}

type searchPageData struct {
	Query   string
	Results []Hit
}

// handleSearchPage is the full results page Enter goes to: the same results as the
// panel, ending with the other two searches.
func (h *Handlers) handleSearchPage(w http.ResponseWriter, r *http.Request) {
	hits, err := h.search(r)
	if err != nil {
		log.Printf("questions: search: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "questions-search.html", "Search questions", searchPageData{Query: r.URL.Query().Get("q"), Results: hits})
}
