package tasks

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/app/format"
	"github.com/stas-comp/comphq/internal/people"
)

// invalidDateMessage is shown when a typed due date isn't a real day.
const invalidDateMessage = "Please type the due date as day/month/year, like 25/09/2026."

// stageLabels are the Board column headings, in display order (SPEC
// gate 2.01).
var stageLabels = map[string]string{
	StageIdea:  "Ideas",
	StageTodo:  "To do",
	StageDoing: "In progress",
	StageDone:  "Done",
}

type boardColumn struct {
	Stage string
	Label string
	Count int
	Tasks []cardView
}

type boardPageData struct {
	CurrentView    string
	Columns        []boardColumn
	People         []people.Person
	Message        string
	FilterMine     bool
	FilterPersonID int64
	FilterQuery    string
	FilterActive   bool
}

// assigneeView adds the display-only initials/colour html/template can't
// compute itself (no registered template funcs — every other section in
// this codebase precomputes view fields in Go instead). ColorClass is a
// CSS class, not an inline style: the app's CSP has no 'unsafe-inline'
// for style-src (or default-src, which style-src falls back to), so an
// inline background-color would be silently dropped rather than applied.
type assigneeView struct {
	Name       string
	Initials   string
	ColorClass string
	Removed    bool
}

// stageOption is one entry in a card's "Move to…" dropdown (SPEC gate
// 2.04).
type stageOption struct {
	Stage string
	Label string
}

// cardView is one Board card (SPEC gate 2.03), plus the move buttons'
// own state (gate 2.04/2.05): PrevID/NextID are the neighbouring card's
// id within this card's own stage, 0 if there isn't one (top/bottom of
// the column, so that button is disabled) — the handler computes these
// once per render from the already-loaded, already-ordered board.
type cardView struct {
	ID        int64
	Title     string
	Size      string
	SizeLabel string
	Stage     string
	DueDate   string
	// DueLabel is the due date as a card shows it ("Wed 23 Sep").
	DueLabel    string
	Overdue     bool
	Done        bool
	Rank        int // 1-based priority number; only To do cards have one (gate 4.16)
	Assignees   []assigneeView
	PrevID      int64
	NextID      int64
	OtherStages []stageOption
}

// MoveUp and MoveDown are the card's two icon-only move buttons (gate
// 4.11): disabled at the top and bottom of their column.
func (c cardView) MoveUp() app.IconButton {
	return app.NewIconButton(app.IconMoveUp, c.Title, c.PrevID == 0)
}

func (c cardView) MoveDown() app.IconButton {
	return app.NewIconButton(app.IconMoveDown, c.Title, c.NextID == 0)
}

func newCardView(t Task, meID int64, today time.Time) cardView {
	views := make([]assigneeView, 0, len(t.Assignees))
	for _, a := range t.Assignees {
		views = append(views, assigneeView{
			Name:       a.Name,
			Initials:   InitialsFor(a.Name),
			ColorClass: AvatarClass(a.PersonID, meID),
			Removed:    a.Removed,
		})
	}
	var otherStages []stageOption
	for _, stage := range Stages {
		if stage == t.Stage {
			continue
		}
		otherStages = append(otherStages, stageOption{Stage: stage, Label: stageLabels[stage]})
	}
	dueLabel := ""
	if due, err := time.Parse("2006-01-02", t.DueDate); err == nil {
		dueLabel = format.DateShort(due, today)
	}
	return cardView{
		ID: t.ID, Title: t.Title, Size: t.Size, SizeLabel: sizeLabels[t.Size], Stage: t.Stage,
		DueDate: t.DueDate, DueLabel: dueLabel, Overdue: t.Overdue, Done: t.Stage == StageDone,
		Assignees: views, OtherStages: otherStages,
	}
}

// redirectTargetFrom reads the shared, allowlisted redirect_to field:
// the move and assign endpoints are each reused from more than one
// page, so a plain (non-JS) form submission needs to land back on
// whichever page it came from — "" means the caller's own default
// (the Board for move, Team for assign).
func redirectTargetFrom(r *http.Request) string {
	switch r.FormValue("redirect_to") {
	case "team":
		return "/tasks/team"
	case "myjobs":
		return "/tasks"
	default:
		return ""
	}
}

func (h *Handlers) handleBoard(w http.ResponseWriter, r *http.Request) {
	h.renderBoard(w, r, http.StatusOK, "")
}

// handleVersion answers with the tasks_version counter (SPEC B4/D-15):
// refresh.js polls this, not the whole board, to notice a change
// happened elsewhere as cheaply as possible.
func (h *Handlers) handleVersion(w http.ResponseWriter, r *http.Request) {
	version, err := h.tasks.Version(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"version": version})
}

// boardFilterFromRequest reads ?person=, ?mine=1 and ?q= (SPEC gate
// 2.07's filter state living in the URL). The Everyone / My tasks / Person
// filter is one drop-down whose value is "" (everyone), "mine" or a
// person's id (gate 4.15); the older ?mine=1 still works. "My tasks"
// takes priority over a chosen person if somehow both are present — they
// express the same kind of filter, and the store only ever filters by
// one id.
func boardFilterFromRequest(r *http.Request) (mine bool, personID int64, query string) {
	mine = r.URL.Query().Get("mine") == "1"
	if v := r.URL.Query().Get("person"); v == "mine" {
		mine = true
	} else if v != "" {
		personID, _ = strconv.ParseInt(v, 10, 64)
	}
	query = r.URL.Query().Get("q")
	return mine, personID, query
}

func (h *Handlers) renderBoard(w http.ResponseWriter, r *http.Request, status int, message string) {
	today := app.Today(h.srv.TestMode)
	mine, personID, query := boardFilterFromRequest(r)

	filter := BoardFilter{Query: query}
	if mine {
		if person, ok := people.FromContext(r.Context()); ok {
			filter.PersonID = person.ID
		}
	} else {
		filter.PersonID = personID
	}

	tasks, err := h.tasks.ListBoard(r.Context(), today, filter)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	byStage := make(map[string][]cardView, len(Stages))
	for _, t := range tasks {
		byStage[t.Stage] = append(byStage[t.Stage], newCardView(t, currentPersonID(r), today))
	}

	columns := make([]boardColumn, 0, len(Stages))
	for _, stage := range Stages {
		cards := byStage[stage]
		for i := range cards {
			if i > 0 {
				cards[i].PrevID = cards[i-1].ID
			}
			if i < len(cards)-1 {
				cards[i].NextID = cards[i+1].ID
			}
			if stage == StageTodo {
				cards[i].Rank = i + 1
			}
		}
		columns = append(columns, boardColumn{Stage: stage, Label: stageLabels[stage], Count: len(cards), Tasks: cards})
	}

	activePeople, err := h.people.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.srv.RenderFrame(w, r, status, "tasks-board.html", "Tasks", boardPageData{
		CurrentView:    "board",
		Columns:        columns,
		People:         activePeople,
		Message:        message,
		FilterMine:     mine,
		FilterPersonID: personID,
		FilterQuery:    query,
		FilterActive:   mine || personID != 0 || query != "",
	})
}

// parsePersonIDs reads a form's repeated person_id values (a multi-select
// of assignees), shared by the add-task and details forms.
func parsePersonIDs(r *http.Request) ([]int64, error) {
	var personIDs []int64
	for _, raw := range r.Form["person_id"] {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, err
		}
		personIDs = append(personIDs, id)
	}
	return personIDs, nil
}

func (h *Handlers) handleCreateTask(w http.ResponseWriter, r *http.Request) {
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

	input := CreateInput{
		Title:     r.FormValue("title"),
		Notes:     r.FormValue("notes"),
		Size:      r.FormValue("size"),
		Stage:     r.FormValue("stage"),
		DueDate:   format.NormaliseDate(r.FormValue("due_date")),
		PersonIDs: personIDs,
	}

	today := app.Today(h.srv.TestMode)
	_, err = h.tasks.Create(r.Context(), input, person.ID, today)
	switch err {
	case nil:
		http.Redirect(w, r, "/tasks/board", http.StatusFound)
	case ErrEmptyTitle:
		h.renderNewTask(w, r, http.StatusOK, "Please type a title.", input, personIDs)
	case ErrInvalidDate:
		h.renderNewTask(w, r, http.StatusOK, invalidDateMessage, input, personIDs)
	case ErrInvalidSize, ErrInvalidStage:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// handleMoveTask serves every button-driven move (SPEC gate 2.04/2.05):
// Move up/down send before_id/after_id computed from the card's current
// neighbours, and Move to… sends a different stage with to_bottom.
func (h *Handlers) handleMoveTask(w http.ResponseWriter, r *http.Request) {
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

	input := MoveInput{Stage: r.FormValue("stage"), ToBottom: r.FormValue("to_bottom") != ""}
	if v := r.FormValue("before_id"); v != "" {
		if input.BeforeID, err = strconv.ParseInt(v, 10, 64); err != nil {
			http.Error(w, "invalid before_id", http.StatusBadRequest)
			return
		}
	}
	if v := r.FormValue("after_id"); v != "" {
		if input.AfterID, err = strconv.ParseInt(v, 10, 64); err != nil {
			http.Error(w, "invalid after_id", http.StatusBadRequest)
			return
		}
	}

	// The Team view's Up next reorder, and My jobs' Start/Done buttons,
	// post to this same endpoint (SPEC B4: "the existing move
	// endpoint"), so a plain (non-JS) form submission needs to land back
	// on whichever page it came from, not always default to the Board —
	// an explicit allowlisted field, not the Referer header, which isn't
	// reliably present.
	redirectTarget := redirectTargetFrom(r)

	today := app.Today(h.srv.TestMode)
	switch err := h.tasks.Move(r.Context(), id, input, person.ID, today); err {
	case nil:
		if redirectTarget != "" {
			http.Redirect(w, r, redirectTarget, http.StatusFound)
			return
		}
		http.Redirect(w, r, "/tasks/board", http.StatusFound)
	case ErrTaskNotFound, ErrNeighborNotFound:
		if redirectTarget != "" {
			// The list changed under us — someone else moved or
			// assigned something first. A fresh redirect is simplest;
			// neither Team nor My jobs carries filter/message state
			// worth preserving across it (unlike the Board).
			http.Redirect(w, r, redirectTarget, http.StatusFound)
			return
		}
		// The board changed under us — someone else moved something
		// first. Just show the current board rather than an error page.
		h.renderBoard(w, r, http.StatusOK, "")
	case ErrInvalidMove, ErrInvalidStage:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
