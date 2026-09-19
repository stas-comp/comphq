package app

import "net/http"

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
