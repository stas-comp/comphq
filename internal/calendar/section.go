// Package calendar will hold events and recurrence (P2-13–P2-17). For now
// it registers only the "Coming soon" placeholder (SPEC gate 1.08).
package calendar

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
)

func Section(srv *app.Server) app.Section {
	return app.Section{
		Nav: &app.NavItem{Label: "Calendar", Path: "/calendar", Icon: "/static/theme/icons/calendar.svg"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /calendar", srv.ComingSoon("Calendar"))
		},
	}
}
