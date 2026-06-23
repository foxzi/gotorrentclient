package web

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"gotorrentclient/utils"
)

// torrentView is the view-model for a single torrent row.
type torrentView struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Percent       float64 `json:"percent"`
	Completed     string  `json:"completed"`
	Length        string  `json:"length"`
	Peers         int     `json:"peers"`
	Uploaded      string  `json:"uploaded"`
	Done          bool    `json:"done"`
	Paused        bool    `json:"paused"`
	Checking      bool    `json:"checking"`
	Status        string  `json:"status"`
	DownloadSpeed string  `json:"download_speed"`
	UploadSpeed   string  `json:"upload_speed"`
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
	data := struct{ Torrents []torrentView }{Torrents: s.torrentViews()}
	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) torrentViews() []torrentView {
	list := s.mgr.List()
	views := make([]torrentView, 0, len(list))
	for _, t := range list {
		views = append(views, torrentView{
			ID:            t.ID,
			Name:          t.Name,
			Percent:       math.Round(t.Percent*10) / 10,
			Completed:     utils.FormatBytes(t.Completed),
			Length:        utils.FormatBytes(t.Length),
			Peers:         t.Peers,
			Uploaded:      utils.FormatBytes(t.Uploaded),
			Done:          t.Done,
			Paused:        t.Paused,
			Checking:      t.Checking,
			Status:        t.Status,
			DownloadSpeed: utils.FormatBytes(t.DownloadSpeed) + "/s",
			UploadSpeed:   utils.FormatBytes(t.UploadSpeed) + "/s",
		})
	}
	return views
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	if err := s.addFromRequest(r); err != nil {
		http.Error(w, "Failed to add torrent: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) addFromRequest(r *http.Request) error {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return err
	}

	var errs []error
	magnet := strings.TrimSpace(r.FormValue("magnet"))
	if magnet != "" {
		if err := s.mgr.AddMagnet(magnet); err != nil {
			errs = append(errs, err)
		}
	}

	if r.MultipartForm != nil {
		for _, header := range r.MultipartForm.File["torrentfile"] {
			if header.Size == 0 {
				continue
			}
			if err := s.addUploadedTorrentFile(header); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

func (s *Server) addUploadedTorrentFile(header *multipart.FileHeader) error {
	file, err := header.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "*.torrent")
	if err != nil {
		return err
	}
	path := tmp.Name()
	defer tmp.Close()
	defer os.Remove(path)

	if _, err := io.Copy(tmp, file); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return s.mgr.AddFile(path)
}

func (s *Server) handleDrop(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id != "" {
		s.mgr.Drop(id)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleAPITorrents(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"torrents": s.torrentViews()})
}

func (s *Server) handleAPIAdd(w http.ResponseWriter, r *http.Request) {
	if err := s.addFromRequest(r); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":    err.Error(),
			"torrents": s.torrentViews(),
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"torrents": s.torrentViews()})
}

func (s *Server) handleAPIDrop(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id != "" {
		s.mgr.Drop(id)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"torrents": s.torrentViews()})
}

func (s *Server) handleAPIPause(w http.ResponseWriter, r *http.Request) {
	if !s.mgr.Pause(r.FormValue("id")) {
		s.writeJSON(w, http.StatusNotFound, map[string]any{
			"error":    "torrent not found",
			"torrents": s.torrentViews(),
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"torrents": s.torrentViews()})
}

func (s *Server) handleAPIResume(w http.ResponseWriter, r *http.Request) {
	if !s.mgr.Resume(r.FormValue("id")) {
		s.writeJSON(w, http.StatusNotFound, map[string]any{
			"error":    "torrent not found",
			"torrents": s.torrentViews(),
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"torrents": s.torrentViews()})
}

func (s *Server) handleAPIVerify(w http.ResponseWriter, r *http.Request) {
	if !s.mgr.Verify(r.FormValue("id")) {
		s.writeJSON(w, http.StatusNotFound, map[string]any{
			"error":    "torrent not found or metadata is not available yet",
			"torrents": s.torrentViews(),
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"torrents": s.torrentViews()})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderLogin(w http.ResponseWriter, errMsg string) {
	data := struct{ Error string }{Error: errMsg}
	if err := s.tmpl.ExecuteTemplate(w, "login.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
