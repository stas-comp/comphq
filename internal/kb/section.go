// Package kb will hold the Knowledge Base (P1-17 onward): categories,
// articles, versions, search and images. For now it registers only the
// "Coming soon" placeholder (SPEC gate 1.08's pattern, applied to the "/"
// redirect target ahead of P1-17).
package kb

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app"
)

func Section(srv *app.Server) app.Section {
	return app.Section{
		MigrationName: "kb", // migrations/kb/0001_search.sql (P1-08)
		Nav:           &app.NavItem{Label: "Knowledge Base", Path: "/kb", Icon: "/static/theme/icons/kb.svg"},
		RegisterRoutes: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /kb", srv.ComingSoon("Knowledge Base"))
		},
	}
}
