package handler

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/asaqe/surge-host/internal/auth"
	"github.com/asaqe/surge-host/internal/config"
	"github.com/asaqe/surge-host/internal/store"
)

// PublicFile represents a publicly listed rule file.
type PublicFile struct {
	User     string
	Name     string
	RawURL   string
	Size     int64
	Modified string
}

// HomeHandler renders the landing page with public file listings.
type HomeHandler struct {
	cfg   *config.Config
	auth  *auth.Service
	store *store.FileStore
	tmpl  *template.Template
}

// NewHomeHandler creates a HomeHandler with parsed templates.
func NewHomeHandler(cfg *config.Config, authSvc *auth.Service, fs *store.FileStore) (*HomeHandler, error) {
	tmpl, err := loadTemplates(cfg.TemplatesDir)
	if err != nil {
		return nil, err
	}
	return &HomeHandler{cfg: cfg, auth: authSvc, store: fs, tmpl: tmpl}, nil
}

// ServeHTTP handles GET /.
func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	records, err := h.store.ListPublic()
	if err != nil {
		slog.Error("list public files failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	files := make([]PublicFile, 0, len(records))
	for _, rec := range records {
		files = append(files, PublicFile{
			User:     rec.Username,
			Name:     rec.Path,
			RawURL:   h.cfg.RawURL(rec.Username, rec.Path),
			Size:     rec.Size,
			Modified: rec.UpdatedAt.Format("2006-01-02 15:04"),
		})
	}

	data := pageData(h.cfg, r, h.auth, map[string]any{
		"Title":     "Surge Host — 多平台代理配置托管",
		"ActiveNav": "home",
		"Files":     files,
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		slog.Error("render index failed", "error", err)
	}
}
