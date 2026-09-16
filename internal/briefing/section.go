// Package briefing will hold the Saturday Briefing (P3-01–P3-03). For now
// it registers only the "Coming soon" placeholder (SPEC gate 1.08).
package briefing

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
)

func Section(srv *app.Server) app.Section {
	return app.Section{
		Nav: &app.NavItem{Label: "Briefing", Path: "/briefing", Icon: "/static/theme/icons/briefing.svg"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /briefing", srv.ComingSoon("Briefing"))
		},
	}
}
