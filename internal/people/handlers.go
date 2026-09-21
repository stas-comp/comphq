package people

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Handlers serves the name picker and holds the person-identity middleware
// (SPEC B4). Section registration (routes, nav, migrations all in one
// place) is generalised in P1-14; for now the caller wires these in
// directly (docs/decisions.md).
type Handlers struct {
	Store *Store
	tmpl  *template.Template
}

// New builds Handlers. tmpl must have who.html parsed into it.
func New(store *Store, tmpl *template.Template) *Handlers {
	return &Handlers{Store: store, tmpl: tmpl}
}

// RegisterRoutes adds the picker routes to mux.
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /who", h.handleWho)
	mux.HandleFunc("POST /who/select", h.handleSelect)
	mux.HandleFunc("POST /who/add", h.handleAdd)
}

type personContextKey struct{}

// FromContext returns the current request's person, once Middleware has
// run.
func FromContext(ctx context.Context) (Person, bool) {
	p, ok := ctx.Value(personContextKey{}).(Person)
	return p, ok
}

// exemptPrefixes are routes the identity middleware never gates: the
// picker itself (or it could never be reached), health/static assets, and
// test-only harness routes.
var exemptPrefixes = []string{"/who", "/healthz", "/static/", "/favicon.ico", "/__test/"}

func isExempt(path string) bool {
	for _, p := range exemptPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// Middleware enforces person identity (SPEC B4): a GET without a valid
// (known, active) person redirects to /who?next=<path>; a POST renders the
// picker directly, with a message, since redirecting a POST would silently
// drop it.
func (h *Handlers) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		id, ok := personIDFromCookie(r)
		var person Person
		if ok {
			var active bool
			var err error
			person, active, err = h.Store.Get(id)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			ok = active
		}

		if !ok {
			if r.Method == http.MethodGet {
				http.Redirect(w, r, "/who?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusFound)
				return
			}
			h.renderPicker(w, safeNext(r.FormValue("next")), "Please pick your name before doing that.")
			return
		}

		setPersonCookie(w, person.ID) // refreshed on every visit (SPEC B4)
		ctx := context.WithValue(r.Context(), personContextKey{}, person)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handlers) handleWho(w http.ResponseWriter, r *http.Request) {
	h.renderPicker(w, safeNext(r.URL.Query().Get("next")), "")
}

func (h *Handlers) handleSelect(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.FormValue("next"))
	idStr := r.FormValue("person_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.renderPicker(w, next, "Please choose a name.")
		return
	}
	person, active, err := h.Store.Get(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !active {
		h.renderPicker(w, next, "That name isn't available any more.")
		return
	}
	setPersonCookie(w, person.ID)
	http.Redirect(w, r, next, http.StatusFound)
}

func (h *Handlers) handleAdd(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.FormValue("next"))
	name := r.FormValue("name")

	person, err := h.Store.Create(name)
	switch err {
	case nil:
		setPersonCookie(w, person.ID)
		http.Redirect(w, r, next, http.StatusFound)
	case ErrEmptyName:
		h.renderPicker(w, next, "Please type a name.")
	case ErrDuplicateName:
		h.renderPicker(w, next, "That name is already taken — pick it from the list, or add a different name.")
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handlers) renderPicker(w http.ResponseWriter, next, message string) {
	people, err := h.Store.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		People  []Person
		Next    string
		Message string
	}{People: people, Next: next, Message: message}
	if err := h.tmpl.ExecuteTemplate(w, "who.html", data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// safeNext only ever returns a local, same-site path: an empty, absent, or
// externally-pointing "next" (e.g. "//evil.example" or "https://...")
// falls back to "/", so the picker can never be made to redirect off-site.
func safeNext(next string) string {
	if next == "" || next[0] != '/' || strings.HasPrefix(next, "//") {
		return "/"
	}
	return next
}
