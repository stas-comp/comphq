package app

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"

	"github.com/stas-comp/comphq/internal/people"
)

// frameData is what layout.html needs to render the sidebar, top bar, and
// the current page's own content (SPEC A4, gate 1.07).
type frameData struct {
	Title       string
	PersonName  string
	ChangeHref  string
	Nav         []navItemData
	BodyContent template.HTML
}

type navItemData struct {
	Label   string
	Path    string
	Icon    string
	Current bool
}

// renderFrame runs contentTemplate (a page's own named template) and
// embeds the result in the shared layout: wordmark, nav, top bar with
// search-box placeholder and "You: name · Change", and the current
// section marked with aria-current (SPEC gate 1.07).
func (s *Server) renderFrame(w http.ResponseWriter, r *http.Request, status int, contentTemplate, title string, pageData any) {
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
			Label:   item.Label,
			Path:    item.Path,
			Icon:    item.Icon,
			Current: item.Path == r.URL.Path,
		})
	}

	data := frameData{
		Title:       title,
		PersonName:  personName,
		ChangeHref:  "/who?next=" + url.QueryEscape(r.URL.RequestURI()),
		Nav:         navItems,
		BodyContent: template.HTML(body.String()), //nolint:gosec // body comes from our own sanitised/static templates, not user input
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
		s.renderFrame(w, r, http.StatusOK, "coming-soon.html", title, struct{ Title string }{Title: title})
	}
}
