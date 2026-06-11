package web

import (
	"math"
	"net/http"
	"os"

	"gotorrentclient/utils"
)

// torrentView is the view-model for a single torrent row.
type torrentView struct {
	ID        string
	Name      string
	Percent   float64
	Completed string
	Length    string
	Peers     int
	Uploaded  string
	Done      bool
}

func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	s.renderLogin(w, "")
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderLogin(w, "Invalid request")
		return
	}
	user := r.FormValue("username")
	pass := r.FormValue("password")
	if !checkCredentials(s.cfg.Username, s.cfg.Password, user, pass) {
		s.renderLogin(w, "Invalid username or password")
		return
	}
	token, err := s.sessions.create()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.sessions.delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	list := s.mgr.List()
	views := make([]torrentView, 0, len(list))
	for _, t := range list {
		views = append(views, torrentView{
			ID:        t.ID,
			Name:      t.Name,
			Percent:   math.Round(t.Percent*10) / 10,
			Completed: utils.FormatBytes(t.Completed),
			Length:    utils.FormatBytes(t.Length),
			Peers:     t.Peers,
			Uploaded:  utils.FormatBytes(t.Uploaded),
			Done:      t.Done,
		})
	}
	data := struct{ Torrents []torrentView }{Torrents: views}
	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	magnet := r.FormValue("magnet")
	if magnet != "" {
		if err := s.mgr.AddMagnet(magnet); err != nil {
			http.Error(w, "Failed to add magnet: "+err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	file, _, err := r.FormFile("torrentfile")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "*.torrent")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer tmp.Close()

	buf := make([]byte, 10<<20)
	n, _ := file.Read(buf)
	if _, err := tmp.Write(buf[:n]); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	path := tmp.Name()
	tmp.Close()

	if err := s.mgr.AddFile(path); err != nil {
		http.Error(w, "Failed to add torrent: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDrop(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id != "" {
		s.mgr.Drop(id)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) renderLogin(w http.ResponseWriter, errMsg string) {
	data := struct{ Error string }{Error: errMsg}
	if err := s.tmpl.ExecuteTemplate(w, "login.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
