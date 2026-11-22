package tasks

import (
	"bufio"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/stas-comp/comphq/internal/app"
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
	Size     string
	Stage    string
	DueDate  string // as typed, day first
	Sizes    []sizeOption
	Stages   []stageOption
	People   []personOption
	// Weekly kept the old tick-box as typed; Repeat and NoticeWeeks keep the
	// Repeats choice and the weeks box as typed when a refused save comes back.
	Weekly      bool
	Repeat      string
	NoticeWeeks string
	// NewSteps, NewArticles and NewPlaces are what was added to the job so far
	// (SPEC B14.3, gates 8.30-8.34): they come back as typed after a refusal.
	// StepsText is the no-script one-per-line box, kept as typed.
	NewSteps    []newStepLine
	StepsText   string
	NewArticles []newArticleLine
	NewPlaces   []newPlaceLine
}

// newStepLine is one step already added in the form. Its three icon buttons are
// the same three a saved step has (move up, move down, remove), but plain
// buttons: they work on the form in the browser and post nothing.
type newStepLine struct{ Text string }

func (l newStepLine) button(kind string) app.IconButton {
	b := app.NewIconButton(kind, "step “"+l.Text+"”", false)
	b.Type = "button"
	return b
}

func (l newStepLine) Up() app.IconButton     { return l.button(app.IconMoveUp) }
func (l newStepLine) Down() app.IconButton   { return l.button(app.IconMoveDown) }
func (l newStepLine) Remove() app.IconButton { return l.button(app.IconRemove) }

func (l newArticleLine) Remove() app.IconButton {
	b := app.NewIconButton(app.IconRemove, "article "+l.Title, false)
	b.Type = "button"
	return b
}

func (l newPlaceLine) Remove() app.IconButton {
	b := app.NewIconButton(app.IconRemove, "place "+l.Label, false)
	b.Type = "button"
	return b
}

// newArticleLine and newPlaceLine are the articles and places already added in
// the form, before the job exists.
type newArticleLine struct {
	ID       int64
	Title    string
	Category string
}

type newPlaceLine struct {
	Name  string
	Path  string
	Label string
}

// newJobExtras is what the Add task form adds beyond the job's own fields.
type newJobExtras struct {
	steps      []string
	articleIDs []int64
	places     []PlaceInput
}

// newJobExtrasFromForm reads the steps (the repeated "step" fields, in order,
// then the lines of the no-script "steps_text" box), the articles (repeated
// "article_id") and the places ("place_name" and "place_path", in pairs) typed
// in the Add task form (SPEC B14.3). The server accepts either input for steps.
func newJobExtrasFromForm(r *http.Request) (newJobExtras, error) {
	var e newJobExtras
	e.steps = append(e.steps, r.PostForm["step"]...)
	lines := bufio.NewScanner(strings.NewReader(r.PostFormValue("steps_text")))
	for lines.Scan() {
		e.steps = append(e.steps, lines.Text())
	}
	for _, raw := range r.PostForm["article_id"] {
		id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return e, err
		}
		e.articleIDs = append(e.articleIDs, id)
	}
	names, paths := r.PostForm["place_name"], r.PostForm["place_path"]
	for i, path := range paths {
		name := ""
		if i < len(names) {
			name = names[i]
		}
		e.places = append(e.places, PlaceInput{Name: name, Path: path})
	}
	return e, nil
}

// newJobRefusal turns a refusal from Create into the plain words the form
// shows, and says whether it was one (gate 8.32).
func newJobRefusal(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrTooManySteps):
		return "A job can have up to 50 steps.", true
	case errors.Is(err, ErrStepTooLong):
		return "A step can be up to 200 characters.", true
	case errors.Is(err, ErrPlacePath), errors.Is(err, ErrPlaceNameLong), errors.Is(err, ErrPlaceDuplicate), errors.Is(err, ErrTooManyPlaces),
		errors.Is(err, ErrArticleLinked), errors.Is(err, ErrTooManyArticles), errors.Is(err, ErrArticleGone), errors.Is(err, ErrArticleArchived):
		return err.Error(), true
	}
	return "", false
}

// handleNewTaskForm serves GET /tasks/new: an empty form, in To do, Medium (gate 7.10).
// With ?fragment=1 it is just the form, for the task window.
func (h *Handlers) handleNewTaskForm(w http.ResponseWriter, r *http.Request) {
	h.renderNewTask(w, r, http.StatusOK, "", CreateInput{Size: "M", Stage: StageTodo}, nil)
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
		stage = StageTodo
	}

	// What was added to the job so far comes back as typed. The articles' words
	// come from the Knowledge Base through the finder, and only the repeated
	// "step" fields are drawn as a list (the no-script box keeps its own text).
	var newSteps []newStepLine
	for _, st := range r.PostForm["step"] {
		if strings.TrimSpace(st) != "" {
			newSteps = append(newSteps, newStepLine{Text: strings.TrimSpace(st)})
		}
	}
	var newArticles []newArticleLine
	known := h.tasks.finderArticles(r.Context(), input.ArticleIDs)
	for _, id := range input.ArticleIDs {
		a := known[id]
		title := a.Title
		if title == "" {
			title = missingArticle
		}
		newArticles = append(newArticles, newArticleLine{ID: id, Title: title, Category: a.Category})
	}
	var newPlaces []newPlaceLine
	for _, p := range input.Places {
		if strings.TrimSpace(p.Name) == "" && strings.TrimSpace(p.Path) == "" {
			continue
		}
		newPlaces = append(newPlaces, newPlaceLine{Name: strings.TrimSpace(p.Name), Path: strings.TrimSpace(p.Path), Label: PlaceDisplayName(p.Name, strings.ReplaceAll(strings.TrimSpace(p.Path), "/", `\`))})
	}

	data := newTaskPageData{
		NewSteps: newSteps, StepsText: r.PostFormValue("steps_text"), NewArticles: newArticles, NewPlaces: newPlaces,
		Message:     message,
		InWindow:    wantsFragment(r),
		Title:       input.Title,
		Notes:       input.Notes,
		Size:        size,
		Stage:       stage,
		DueDate:     format.DayFirst(input.DueDate),
		Sizes:       sizeOptions,
		Stages:      stages,
		People:      options,
		Weekly:      input.Weekly,
		Repeat:      repeatChoice(input),
		NoticeWeeks: noticeWeeksTyped(r, input),
	}
	if data.InWindow {
		h.srv.RenderPartial(w, status, "tasks-new-form", data)
		return
	}
	h.srv.RenderFrame(w, r, status, "tasks-new.html", "Add task", data)
}

// repeatChoice is the Repeats choice a form comes back with.
func repeatChoice(input CreateInput) string {
	switch {
	case input.Repeat == repeatYearly || input.Repeat == repeatWeekly:
		return input.Repeat
	case input.Weekly:
		return repeatWeekly
	}
	return repeatNone
}

// noticeWeeksTyped is the weeks box as typed, or the default.
func noticeWeeksTyped(r *http.Request, input CreateInput) string {
	if v := strings.TrimSpace(r.PostFormValue("notice_weeks")); v != "" {
		return v
	}
	if input.NoticeWeeks > 0 {
		return strconv.Itoa(input.NoticeWeeks)
	}
	return strconv.Itoa(defaultNoticeWeeks)
}
