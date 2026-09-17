package app

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/db"
	"github.com/stas-comp/comphq/internal/people"
)

// frameData is what layout.html needs to render the sidebar, top bar, and
// the current page's own content (SPEC A4, gate 1.07).
type frameData struct {
	Title            string
	PersonName       string
	ChangeHref       string
	Nav              []navItemData
	BodyContent      template.HTML
	ShowBackupBanner bool
}

type navItemData struct {
	Label   string
	Path    string
	Icon    string
	Current bool
}

// RenderFrame runs contentTemplate (a page's own named template) and
// embeds the result in the shared layout: wordmark, nav, top bar with
// search-box placeholder and "You: name · Change", and the current
// section marked with aria-current (SPEC gate 1.07). Exported so section
// packages can render their own pages the same way ComingSoon does.
func (s *Server) RenderFrame(w http.ResponseWriter, r *http.Request, status int, contentTemplate, title string, pageData any) {
	var body bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&body, contentTemplate, pageData); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	personName := ""
	if p, ok := people.FromContext(r.Context()); ok {
		personName = p.Name
	}

	navItems := make([]navItemData, 0, len(s.registry.NavItems()))
	for _, item := range s.registry.NavItems() {
		navItems = append(navItems, navItemData{
			Label: item.Label,
			Path:  item.Path,
			Icon:  item.Icon,
			// A sub-page (e.g. "/settings/people") still marks its
			// section's nav item current, not just an exact match.
			Current: r.URL.Path == item.Path || strings.HasPrefix(r.URL.Path, item.Path+"/"),
		})
	}

	showBackupBanner := false
	if status, err := db.GetBackupStatus(s.DB); err != nil {
		log.Printf("frame: reading backup status: %v", err)
	} else {
		showBackupBanner = status.Stale(time.Now())
	}

	data := frameData{
		Title:            title,
		PersonName:       personName,
		ChangeHref:       "/who?next=" + url.QueryEscape(r.URL.RequestURI()),
		Nav:              navItems,
		BodyContent:      template.HTML(body.String()), //nolint:gosec // body comes from our own sanitised/static templates, not user input
		ShowBackupBanner: showBackupBanner,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// ComingSoon shows the shared placeholder for a section that isn't built
// yet (SPEC gate 1.08), inside the normal frame. Removed for a section
// the moment it's actually built (SPEC B7 rule 5's one allowed test
// change).
func (s *Server) ComingSoon(title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.RenderFrame(w, r, http.StatusOK, "coming-soon.html", title, struct{ Title string }{Title: title})
	}
}
