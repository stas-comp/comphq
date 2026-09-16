package app

import (
	"database/sql"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/stas-comp/comphq"
)

// Server holds what request handlers need: the database and the parsed
// frame templates. Later tasks add the section registry (P1-14).
type Server struct {
	DB       *sql.DB
	Version  string
	TestMode bool
	tmpl     *template.Template
	static   http.Handler
}

// NewServer parses the embedded templates and builds a Server. testMode
// gates test-only routes (SPEC P1-09): never true in deploy/truenas.yaml.
func NewServer(sqlDB *sql.DB, version string, testMode bool) (*Server, error) {
	tmpl, err := template.ParseFS(comphq.Templates, "web/templates/app/*.html")
	if err != nil {
		return nil, err
	}

	staticFS, err := fs.Sub(comphq.Static, "web/static")
	if err != nil {
		return nil, err
	}

	return &Server{
		DB:       sqlDB,
		Version:  version,
		TestMode: testMode,
		tmpl:     tmpl,
		static:   http.FileServerFS(staticFS),
	}, nil
}

// Routes builds the HTTP handler. The frame route is a catch-all for now;
// P1-14 adds the section registry, navigation, and a real 404 page.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.Handle("GET /static/", http.StripPrefix("/static/", s.static))
	if s.TestMode {
		mux.HandleFunc("GET /__test/editor", s.handleTestEditor)
		mux.HandleFunc("GET /__test/egress", s.handleTestEgress)
	}
	mux.HandleFunc("GET /", s.handleFrame)
	return securityHeaders(mux)
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

func (s *Server) handleFrame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "frame.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
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
