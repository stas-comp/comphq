// Package calendar holds events and recurrence (SPEC A7).
package calendar

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
)

type Handlers struct {
	srv   *app.Server
	store *Store
}

func Section(srv *app.Server) app.Section {
	h := &Handlers{srv: srv, store: &Store{DB: srv.DB}}
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
