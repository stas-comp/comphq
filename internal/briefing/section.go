package briefing

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/app/format"
	"github.com/stas-comp/comphq/internal/people"
)

// TaskSource is the small query interface the Briefing needs from
// Tasks, injected at registration rather than imported (SPEC B2, D-53).
// cmd/comphq adapts *tasks.Store to it.
type TaskSource interface {
	// Tasks returns every unfinished, unremoved task due on or before
	// the given date, overdue included, with its people.
	Tasks(ctx context.Context, dueOnOrBefore time.Time) ([]Task, error)
	// Version is the tasks change counter (SPEC B4's refresh design).
	Version(ctx context.Context) (int64, error)
}

// EventSource is the small query interface the Briefing needs from the
// Calendar, injected the same way.
type EventSource interface {
	// Occurrences returns every occurrence overlapping [from, to],
	// exceptions already applied.
	Occurrences(ctx context.Context, from, to time.Time) ([]Occurrence, error)
	// UpcomingWithNotice returns each series' next occurrence starting
	// after `after` whose show-ahead time has started by `saturday`.
	UpcomingWithNotice(ctx context.Context, after, saturday time.Time) ([]Occurrence, error)
	// Version is the calendar change counter.
	Version(ctx context.Context) (int64, error)
}

// Handlers serves the Briefing.
type Handlers struct {
	srv    *app.Server
	tasks  TaskSource
	events EventSource
}

// Section registers the Briefing as the first screen: both "/" and
// "/briefing" render it (SPEC gate 3.01).
func Section(srv *app.Server, tasks TaskSource, events EventSource) app.Section {
	h := &Handlers{srv: srv, tasks: tasks, events: events}
	return app.Section{
		Nav: &app.NavItem{
			Label:         "Briefing",
			Path:          "/briefing",
			Icon:          "briefing",
			AlsoCurrentAt: []string{"/"},
		},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /{$}", h.handleBriefing)
			mux.HandleFunc("GET /briefing", h.handleBriefing)
			mux.HandleFunc("GET /briefing/version", h.handleVersion)
		},
	}
}

type pageData struct {
	Briefing Briefing
	// The page's date heading (gate 4.44) is one h1 whose text is the SPEC
	// gate 3.02 headline ("Saturday 19 September — today's briefing"): the
	// day, the date in the deeper accent, and a small tag drawn above them.
	// TagFirst is true when the tag reads first ("Briefing for …").
	HeadlineDay  string
	HeadlineDate string
	HeadlineTag  string
	TagFirst     bool
	MustDo       string // section title
	FridayLabel  string // "Fri 25 Sep", the end of This week
	Path         string // this page's own path, for the Just mine switch
	Mine         bool
	MeID         int64
}

func (h *Handlers) handleBriefing(w http.ResponseWriter, r *http.Request) {
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "not signed in", http.StatusForbidden)
		return
	}

	today := app.Today(h.srv.TestMode)
	saturday := SaturdayFor(today)
	friday := saturday.AddDate(0, 0, 6)

	tasks, err := h.tasks.Tasks(r.Context(), friday)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	week, err := h.events.Occurrences(r.Context(), today, friday)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	upcoming, err := h.events.UpcomingWithNotice(r.Context(), friday, saturday)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	b := Build(Input{Today: today, PersonID: person.ID, Tasks: tasks, ThisWeek: week, Upcoming: upcoming})
	mine := r.URL.Query().Get("mine") == "1"
	if mine {
		b = b.JustMine()
	}

	// Every person on a card is drawn as their coloured circle, the same
	// colour as on every other screen (gates 4.08, 4.46).
	for i := range b.MustDo {
		for j := range b.MustDo[i].People {
			p := &b.MustDo[i].People[j]
			p.Initials = format.Initials(p.Name)
			p.ColorClass = people.AvatarClass(p.ID, person.ID)
		}
	}

	data := pageData{
		Briefing:     b,
		Path:         r.URL.Path,
		Mine:         mine,
		MeID:         person.ID,
		HeadlineDay:  saturday.Format("Monday"),
		HeadlineDate: saturday.Format("2 January"),
		FridayLabel:  friday.Format("Mon 2 Jan"),
	}
	if b.IsToday {
		data.HeadlineTag = "today's briefing"
		data.MustDo = "Must be done today"
	} else {
		data.HeadlineTag = "Briefing for"
		data.TagFirst = true
		data.MustDo = "Must be done this Saturday"
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "briefing.html", "Briefing", data)
}

// handleVersion answers with one number that changes whenever either
// tasks or the calendar does (SPEC gate 3.12): refresh.js polls this,
// not the whole page. Each counter only ever goes up, so their sum
// does too, and any single change moves it.
func (h *Handlers) handleVersion(w http.ResponseWriter, r *http.Request) {
	tasksVersion, err := h.tasks.Version(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	calendarVersion, err := h.events.Version(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"version": tasksVersion + calendarVersion})
}
