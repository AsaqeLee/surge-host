package handler

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/asaqe/surge-host/internal/auth"
	"github.com/asaqe/surge-host/internal/config"
	"github.com/asaqe/surge-host/pkg/safepath"
)

// PageHandler renders HTML management pages.
type PageHandler struct {
	cfg  *config.Config
	auth *auth.Service
	tmpl *template.Template
}

// NewPageHandler creates a PageHandler.
func NewPageHandler(cfg *config.Config, authSvc *auth.Service) (*PageHandler, error) {
	tmpl, err := loadTemplates(cfg.TemplatesDir)
	if err != nil {
		return nil, err
	}
	return &PageHandler{cfg: cfg, auth: authSvc, tmpl: tmpl}, nil
}

// Upload renders the file upload page.
func (h *PageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "upload.html", map[string]any{
		"Title":     "Upload — Rules",
		"ActiveNav": "upload",
	})
}

// Files renders the file management page.
func (h *PageHandler) Files(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "files.html", map[string]any{
		"Title":     "Manage — Rules",
		"ActiveNav": "files",
	})
}

// Edit renders the online editor page.
func (h *PageHandler) Edit(w http.ResponseWriter, r *http.Request) {
	filePath, err := safepath.PrepareUserPath(h.cfg.AdminUser, r.PathValue("path"))
	if err != nil {
		http.Error(w, "invalid file path", http.StatusBadRequest)
		return
	}
	h.render(w, r, "edit.html", map[string]any{
		"Title":     "Edit — " + filePath,
		"ActiveNav": "files",
		"FilePath":  filePath,
	})
}

func (h *PageHandler) render(w http.ResponseWriter, r *http.Request, name string, extra map[string]any) {
	data := pageData(h.cfg, r, h.auth, extra)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("render page failed", "template", name, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}