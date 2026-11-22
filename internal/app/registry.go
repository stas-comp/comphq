package app

import (
	"encoding/json"
	"net/http"
)

// NavItem is one entry in the sidebar (SPEC A4).
type NavItem struct {
	Label string
	Path  string
	// Icon is the name of a line icon in web/static/theme/icons/, without
	// its extension: "kb" draws kb.svg (SPEC B9.4). theme.css has one
	// `.icon-<name>` class per icon.
	Icon string
	// AlsoCurrentAt lists other exact paths that mark this item current
	// too (the Briefing is also the home page, "/").
	AlsoCurrentAt []string
	// Search is what the top bar's search box does on this section's pages
	// (gate 7.20, D-94). Nil means the Knowledge Base's article search.
	Search *SearchSetting
}

// SearchLink is a link to another kind of search, offered under the results.
type SearchLink struct {
	Label string `json:"label"` // "Search articles instead"
	Href  string `json:"href"`  // that search's results page; the script adds ?q=
}

// SearchSetting is one kind of top-bar search: where the form goes, where the
// live panel fetches from, what the box says, and the links to the other kinds
// of search offered under the results.
type SearchSetting struct {
	Kind    string // "articles", "jobs" or "questions": which results the script draws
	Action  string // the full results page
	JSONURL string // the live panel's source
	Label   string // the box's label and placeholder
	// Insteads are the other searches (gates 7.24, 8.78): every section's results
	// end with links to the other two.
	Insteads []SearchLink
}

// InsteadsJSON is Insteads for the search box's data attribute ("" for none).
func (s SearchSetting) InsteadsJSON() string {
	if len(s.Insteads) == 0 {
		return ""
	}
	b, err := json.Marshal(s.Insteads)
	if err != nil {
		return ""
	}
	return string(b)
}

// The three searches' own links, so each section names the others in one place.
var (
	SearchArticlesInstead  = SearchLink{Label: "Search articles instead", Href: "/kb/search"}
	SearchJobsInstead      = SearchLink{Label: "Search jobs instead", Href: "/tasks/search"}
	SearchQuestionsInstead = SearchLink{Label: "Search questions instead", Href: "/questions/search"}
)

// ArticleSearch is the top bar's default: the Knowledge Base's search.
var ArticleSearch = SearchSetting{
	Kind: "articles", Action: "/kb/search", JSONURL: "/kb/search.json", Label: "Search articles",
	// No Insteads: the article panel's foot is gate 4.42's, unchanged. "Search
	// questions instead" is on the full results page (gate 8.78).
}

// Section is what SPEC B2 means by "each section registers its routes,
// navigation entry ... with app": a package exposes one function
// returning a Section, and main.go/server.go add one registration line.
// Migrations are loaded by name (MigrationName) in cmd/comphq/main.go,
// since the migration runner works from the embedded filesystem rather
// than through the section itself.
type Section struct {
	MigrationName  string   // "" if this section has no migrations
	Nav            *NavItem // nil if this section has no sidebar entry
	RegisterRoutes func(mux *http.ServeMux)
}

// Registry collects every section the app serves, in registration order
// (which is also sidebar order).
type Registry struct {
	sections []Section
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Add(s Section) {
	r.sections = append(r.sections, s)
}

func (r *Registry) Sections() []Section {
	return r.sections
}

// NavItems returns every section's sidebar entry, in registration order.
func (r *Registry) NavItems() []NavItem {
	var items []NavItem
	for _, s := range r.sections {
		if s.Nav != nil {
			items = append(items, *s.Nav)
		}
	}
	return items
}

// MigrationNames returns every section's migration directory name, for
// cmd/comphq/main.go to run in order.
func (r *Registry) MigrationNames() []string {
	var names []string
	for _, s := range r.sections {
		if s.MigrationName != "" {
			names = append(names, s.MigrationName)
		}
	}
	return names
}
