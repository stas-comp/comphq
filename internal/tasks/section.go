// Package tasks will hold the Board, Team view and My jobs (P2-01–P2-12).
// For now it registers only the "Coming soon" placeholder (SPEC gate 1.08).
package tasks

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
)

func Section(srv *app.Server) app.Section {
	return app.Section{
		Nav: &app.NavItem{Label: "Tasks", Path: "/tasks", Icon: "/static/theme/icons/tasks.svg"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /tasks", srv.ComingSoon("Tasks"))
		},
	}
}
