package app

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
	"github.com/stas-comp/comphq/internal/people"
)

// Server holds what request handlers need: the database, the parsed
// templates, and the section registry (SPEC B2).
type Server struct {
	DB       *sql.DB
	Version  string
	DataDir  string // SPEC B3: uploaded images live under DataDir/images/
	TestMode bool
	tmpl     *template.Template
	static   http.Handler
	people   *people.Handlers
	registry *Registry
}

// NewServer parses the embedded templates and builds a Server, with the
// "app" and "people" sections already registered. Callers add every other
// section with Registry().Add(...) before calling Routes() (cmd/comphq
// imports all of them; internal/app can't, since a section imports
// internal/app itself — see docs/decisions.md D-14). testMode gates
// test-only routes (SPEC P1-09): never true in deploy/truenas.yaml.
func NewServer(sqlDB *sql.DB, version, dataDir string, testMode bool) (*Server, error) {
	tmpl, err := template.ParseFS(comphq.Templates,
		"web/templates/app/*.html",
		"web/templates/people/*.html",
		"web/templates/settings/*.html",
		"web/templates/kb/*.html",
		"web/templates/tasks/*.html",
		"web/templates/calendar/*.html",
		"web/templates/briefing/*.html",
	)
	if err != nil {
		return nil, err
	}

	staticFS, err := fs.Sub(comphq.Static, "web/static")
	if err != nil {
		return nil, err
	}

	srv := &Server{
		DB:       sqlDB,
		Version:  version,
		DataDir:  dataDir,
		TestMode: testMode,
		tmpl:     tmpl,
		static:   http.FileServerFS(staticFS),
		people:   people.New(&people.Store{DB: sqlDB}, tmpl),
	}

	srv.registry = NewRegistry()
	srv.registry.Add(Section{MigrationName: "app"})
	srv.registry.Add(Section{MigrationName: "people", RegisterRoutes: srv.people.RegisterRoutes})

	return srv, nil
}

// Registry lets the caller (cmd/comphq) add every other section before
// Routes is called.
func (s *Server) Registry() *Registry {
	return s.registry
}

// PeopleStore gives other sections (Settings → People) the same store
// instance the picker and identity middleware use, so a rename or removal
// is visible everywhere immediately.
func (s *Server) PeopleStore() *people.Store {
	return s.people.Store
}

// Routes builds the HTTP handler: health, static assets, test-only
// routes, every registered section, the "/" → "/kb" redirect (SPEC
// P1-14: "in Phase 1"; the Briefing from P3-02), and a 404 fallback
// inside the normal frame (SPEC gate 1.11).
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.Handle("GET /static/", http.StripPrefix("/static/", s.static))
	mux.HandleFunc("GET /favicon.ico", s.handleFavicon)
	if s.TestMode {
		mux.HandleFunc("GET /__test/editor", s.handleTestEditor)
		mux.HandleFunc("GET /__test/parts", s.handleTestParts)
		mux.HandleFunc("GET /__test/egress", s.handleTestEgress)
		mux.HandleFunc("GET /__test/routes", s.handleTestRoutes)
		mux.HandleFunc("POST /__test/people/deactivate", s.handleTestDeactivatePerson)
		mux.HandleFunc("POST /__test/kb/seed-article", s.handleTestSeedKBArticle)
		mux.HandleFunc("POST /__test/backups/set-last-backup-at", s.handleTestSetLastBackupAt)
		mux.HandleFunc("POST /__test/tasks/set-done-at", s.handleTestSetTaskDoneAt)
	}

	for _, section := range s.registry.Sections() {
		if section.RegisterRoutes != nil {
			section.RegisterRoutes(mux)
		}
	}

	mux.HandleFunc("GET /", s.handleNotFound)

	// Order: security headers outermost, then the CSRF-style origin check
	// (SPEC B4 request safety), then person identity (SPEC B4 person
	// identity) — a request needs to clear the origin check before the
	// identity middleware ever renders the picker for it.
	return securityHeaders(OriginCheck(s.people.Middleware(mux)))
}

// securityHeaders sets the headers SPEC B4 requires on every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	s.RenderFrame(w, r, http.StatusNotFound, "404.html", "Page not found", nil)
}

// handleTestRoutes lists every path that renders inside the normal frame,
// for the gate 1.07 E2E test ("frame elements on every registered
// route"). /who is deliberately excluded: it's a full-screen page outside
// the frame (SPEC A4).
func (s *Server) handleTestRoutes(w http.ResponseWriter, r *http.Request) {
	var paths []string
	for _, section := range s.registry.Sections() {
		if section.Nav != nil && section.RegisterRoutes != nil {
			paths = append(paths, section.Nav.Path)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paths)
}

// handleTestEditor serves the P1-09 editor spike harness: proves the
// vendored TipTap bundle runs, with every toolbar feature, under the same
// CSP the real app serves.
func (s *Server) handleTestEditor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "test-editor.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// handleTestEgress proves the container truly has no internet access
// (SPEC P1-12 offline test): it tries one real outbound request and
// reports whether it got through. deploy/test/offline.override.yaml is
// the only place that sets COMPHQ_TEST_MODE=1 on a running container.
func (s *Server) handleTestEgress(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://example.com")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("egress blocked: " + err.Error()))
		return
	}
	resp.Body.Close()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("egress reachable"))
}

// handleTestDeactivatePerson calls the same store helper Settings → People
// (P1-15) will use to remove a person, so gate 1.06 ("if the name chosen on
// a computer is removed, that computer shows the picker next time") is
// testable before that page exists.
func (s *Server) handleTestDeactivatePerson(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.FormValue("person_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid person_id", http.StatusBadRequest)
		return
	}
	if err := s.people.Store.Deactivate(id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleTestSeedKBArticle inserts one minimal published article, so the
// category delete-refusal gate (1.12) and the home page's tile counts
// (1.13) are E2E-testable before the real editor/publish flow exists
// (P1-19).
func (s *Server) handleTestSeedKBArticle(w http.ResponseWriter, r *http.Request) {
	categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid category_id", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.DB.Exec(
		`INSERT INTO kb_articles (category_id, title, status, created_at, updated_at) VALUES (?, ?, 'published', ?, ?)`,
		categoryID, "Test article", now, now,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleTestSetLastBackupAt lets an E2E test move the recorded backup
// time without waiting on the real scheduler (SPEC gate 1.35's own test
// plan: "set last_backup_at to 3 days ago (test-mode helper)"). days_ago
// may be fractional (e.g. "2.1") and negative values are rejected only
// implicitly by AddDate's normal arithmetic — a caller wanting "fresh"
// passes "0".
func (s *Server) handleTestSetLastBackupAt(w http.ResponseWriter, r *http.Request) {
	daysAgo, err := strconv.ParseFloat(r.FormValue("days_ago"), 64)
	if err != nil {
		http.Error(w, "invalid days_ago", http.StatusBadRequest)
		return
	}
	at := time.Now().Add(-time.Duration(daysAgo * float64(24*time.Hour)))
	if err := db.SetLastBackupAtForTest(s.DB, at); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleTestSetTaskDoneAt lets an E2E test move a Done task's done_at
// into the past without waiting on the real 14-day retention window
// (SPEC gate 2.08), the same days_ago convention as
// handleTestSetLastBackupAt. Raw SQL against the shared *sql.DB, not
// the tasks package's own Store, since internal/app can't import a
// section — a section imports internal/app itself (D-14) — the same
// reason handleTestSeedKBArticle above talks to kb_articles directly.
func (s *Server) handleTestSetTaskDoneAt(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(r.FormValue("task_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid task_id", http.StatusBadRequest)
		return
	}
	daysAgo, err := strconv.ParseFloat(r.FormValue("days_ago"), 64)
	if err != nil {
		http.Error(w, "invalid days_ago", http.StatusBadRequest)
		return
	}
	at := time.Now().Add(-time.Duration(daysAgo * float64(24*time.Hour)))
	if _, err := s.DB.Exec(`UPDATE tasks SET done_at = ? WHERE id = ?`, at.UTC().Format(time.RFC3339), taskID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleHealthz answers 200 only once the database answers SELECT 1.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	var one int
	if err := s.DB.QueryRowContext(r.Context(), "SELECT 1").Scan(&one); err != nil || one != 1 {
		http.Error(w, "unhealthy", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
