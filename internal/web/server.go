package web

import (
	"embed"
	"html/template"
	"net/http"

	"gotorrentclient/internal/config"
	"gotorrentclient/internal/torrentmgr"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server is the web UI server.
type Server struct {
	cfg      config.Config
	mgr      *torrentmgr.Manager
	sessions *sessionStore
	tmpl     *template.Template
}

// NewServer creates a Server and parses embedded templates.
func NewServer(cfg config.Config, mgr *torrentmgr.Manager) (*Server, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:      cfg,
		mgr:      mgr,
		sessions: newSessionStore(),
		tmpl:     tmpl,
	}, nil
}

// Routes returns the HTTP mux for the web UI.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	static := http.FileServerFS(staticFS)
	mux.Handle("GET /static/", static)

	mux.HandleFunc("GET /login", s.handleLoginGet)
	mux.HandleFunc("POST /login", s.handleLoginPost)
	mux.HandleFunc("POST /logout", s.handleLogout)

	mux.HandleFunc("GET /", s.authMiddleware(s.handleIndex))
	mux.HandleFunc("POST /add", s.authMiddleware(s.handleAdd))
	mux.HandleFunc("POST /drop", s.authMiddleware(s.handleDrop))

	return mux
}

// authMiddleware redirects to /login if no valid session cookie is present.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil || !s.sessions.valid(c.Value) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	return http.ListenAndServe(s.cfg.Listen, s.Routes())
}
