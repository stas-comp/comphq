package app

import (
	"database/sql"
	"html/template"
	"net/http"

	"github.com/stas-comp/comphq"
)

// Server holds what request handlers need: the database and the parsed
// frame templates. Later tasks add the section registry (P1-14).
type Server struct {
	DB      *sql.DB
	Version string
	tmpl    *template.Template
}

// NewServer parses the embedded templates and builds a Server.
func NewServer(sqlDB *sql.DB, version string) (*Server, error) {
	tmpl, err := template.ParseFS(comphq.Templates, "web/templates/app/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{DB: sqlDB, Version: version, tmpl: tmpl}, nil
}

// Routes builds the HTTP handler. The frame route is a catch-all for now;
// P1-14 adds the section registry, navigation, and a real 404 page.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /", s.handleFrame)
	return mux
}

func (s *Server) handleFrame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "frame.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
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
