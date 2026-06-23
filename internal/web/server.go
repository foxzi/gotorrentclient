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
	mgr      torrentManager
	sessions *sessionStore
	tmpl     *template.Template
}

type torrentManager interface {
	AddMagnet(string) error
	AddFile(string) error
	List() []torrentmgr.TorrentInfo
	Pause(string) bool
	Resume(string) bool
	Verify(string) bool
	Drop(string) bool
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

// authEnabled reports whether authentication is configured.
func (s *Server) authEnabled() bool {
	return s.cfg.Username != "" && s.cfg.Password != ""
}

// Routes returns the HTTP mux for the web UI.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	static := http.FileServerFS(staticFS)
	mux.Handle("GET /static/", static)

	if s.authEnabled() {
		mux.HandleFunc("GET /login", s.handleLoginGet)
		mux.HandleFunc("POST /login", s.handleLoginPost)
		mux.HandleFunc("POST /logout", s.handleLogout)
	}

	mux.HandleFunc("GET /", s.authMiddleware(s.handleIndex))
	mux.HandleFunc("POST /add", s.authMiddleware(s.handleAdd))
	mux.HandleFunc("POST /drop", s.authMiddleware(s.handleDrop))
	mux.HandleFunc("GET /api/torrents", s.authMiddleware(s.handleAPITorrents))
	mux.HandleFunc("POST /api/add", s.authMiddleware(s.handleAPIAdd))
	mux.HandleFunc("POST /api/pause", s.authMiddleware(s.handleAPIPause))
	mux.HandleFunc("POST /api/resume", s.authMiddleware(s.handleAPIResume))
	mux.HandleFunc("POST /api/verify", s.authMiddleware(s.handleAPIVerify))
	mux.HandleFunc("POST /api/drop", s.authMiddleware(s.handleAPIDrop))

	return mux
}

// authMiddleware checks session cookie when auth is enabled; passes through otherwise.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authEnabled() {
			next(w, r)
			return
		}
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
