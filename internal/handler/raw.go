package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/asaqe/surge-host/internal/config"
	"github.com/asaqe/surge-host/pkg/safepath"
)

// RawHandler serves plain-text Surge config files at /raw/{user}/{path...}.
type RawHandler struct {
	cfg *config.Config
}

// NewRawHandler creates a RawHandler.
func NewRawHandler(cfg *config.Config) *RawHandler {
	return &RawHandler{cfg: cfg}
}

// ServeHTTP handles GET /raw/{user}/{path...}.
func (h *RawHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.PathValue("user")
	filePath := r.PathValue("path")

	if user == "" || filePath == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if strings.Contains(user, "/") || strings.Contains(user, "..") {
		http.Error(w, "Invalid user", http.StatusBadRequest)
		return
	}

	if !safepath.AllowedExtension(filePath, h.cfg.AllowedExtensions) {
		http.Error(w, "File type not allowed", http.StatusForbidden)
		return
	}

	userDir := h.cfg.UserDir(user)
	fullPath, err := safepath.JoinSafe(userDir, filePath)
	if err != nil {
		slog.Warn("raw path rejected", "user", user, "path", filePath, "error", err)
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		slog.Error("raw stat failed", "path", fullPath, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if info.IsDir() {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if info.Size() > h.cfg.MaxFileSize {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	f, err := os.Open(fullPath)
	if err != nil {
		slog.Error("raw open failed", "path", fullPath, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	if _, err := io.Copy(w, f); err != nil {
		slog.Error("raw write failed", "path", fullPath, "error", err)
	}
}