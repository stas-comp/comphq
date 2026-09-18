package briefing

import (
	"context"
	"net/http"
	"time"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/people"
)

// TaskSource is the small query interface the Briefing needs from
// Tasks, injected at registration rather than imported (SPEC B2, D-53).
// cmd/comphq adapts *tasks.Store to it.
type TaskSource interface {
	// Tasks returns every unfinished, unremoved task due on or before
	// the given date, overdue included, with its people.
	Tasks(ctx context.Context, dueOnOrBefore time.Time) ([]Task, error)
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
			Icon:          "/static/theme/icons/briefing.svg",
			AlsoCurrentAt: []string{"/"},
		},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /{$}", h.handleBriefing)
			mux.HandleFunc("GET /briefing", h.handleBriefing)
		},
	}
}

type pageData struct {
	Briefing Briefing
	Headline string
	MustDo   string // section title
	Path     string // this page's own path, for the Just mine switch
	Mine     bool
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

	data := pageData{Briefing: b, Path: r.URL.Path, Mine: mine}
	day := saturday.Format("Monday 2 January")
	if b.IsToday {
		data.Headline = day + " — today's briefing"
		data.MustDo = "Must be done today"
	} else {
		data.Headline = "Briefing for " + day
		data.MustDo = "Must be done this Saturday"
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "briefing.html", "Briefing", data)
}
