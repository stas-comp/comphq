// Package calendar holds events and recurrence (SPEC A7).
package calendar

import (
	"context"
	"net/http"
	"time"

	"github.com/stas-comp/comphq/internal/app"
)

// DueTask is one task deadline chip's own data (SPEC gate 2.19) — a
// small, calendar-owned shape, not tasks.DueTask itself, so this
// package never needs to import internal/tasks (SPEC B2: "sections
// never import each other's internals").
type DueTask struct {
	ID      int64
	Title   string
	DueDate string
}

// DueTasksSource is the small query interface Calendar needs from
// Tasks, injected at registration rather than imported directly (SPEC
// B2). *tasks.Store doesn't implement this itself (it returns its own
// []tasks.DueTask) — cmd/comphq wraps it in a small adapter, the one
// place allowed to know about every section.
type DueTasksSource interface {
	DueTasks(ctx context.Context, from, to time.Time) ([]DueTask, error)
}

type Handlers struct {
	srv      *app.Server
	store    *Store
	dueTasks DueTasksSource
}

func Section(srv *app.Server, dueTasks DueTasksSource) app.Section {
	h := &Handlers{srv: srv, store: &Store{DB: srv.DB}, dueTasks: dueTasks}
	return app.Section{
		MigrationName: "calendar",
		Nav:           &app.NavItem{Label: "Calendar", Path: "/calendar", Icon: "/static/theme/icons/calendar.svg"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /calendar", h.handleMonth)
			mux.HandleFunc("GET /calendar/list", h.handleList)
			mux.HandleFunc("GET /calendar/new", h.handleNewEventForm)
			mux.HandleFunc("GET /calendar/removed", h.handleRemovedEvents)
			mux.HandleFunc("POST /calendar/events", h.handleCreateEvent)
			mux.HandleFunc("GET /calendar/events/{id}", h.handleEventDetails)
			mux.HandleFunc("POST /calendar/events/{id}", h.handleUpdateEvent)
			mux.HandleFunc("POST /calendar/events/{id}/remove", h.handleRemoveEvent)
			mux.HandleFunc("POST /calendar/events/{id}/restore", h.handleRestoreEvent)
			mux.HandleFunc("GET /calendar/events/{id}/occurrence", h.handleOccurrenceForm)
			mux.HandleFunc("POST /calendar/events/{id}/occurrence", h.handleSetOccurrence)
			mux.HandleFunc("POST /calendar/events/{id}/occurrence/cancel", h.handleCancelOccurrence)
		},
	}
}
