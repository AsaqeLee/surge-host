package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/asaqe/surge-host/internal/auth"
	"github.com/asaqe/surge-host/pkg/response"
)

// History handles GET /api/git/log/{path...}.
func (h *FileAPIHandler) History(w http.ResponseWriter, r *http.Request) {
	username, _ := auth.UsernameFromContext(r.Context())
	relPath := r.PathValue("path")

	if !h.store.VCS().Enabled() {
		response.Error(w, http.StatusServiceUnavailable, "git versioning is disabled")
		return
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	commits, err := h.store.VCS().Log(username, relPath, limit)
	if err != nil {
		slog.Error("git log failed", "user", username, "path", relPath, "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to read history")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"path":    relPath,
		"commits": commits,
		"raw_url": h.cfg.RawURL(username, relPath),
	})
}

// ShowVersion handles GET /api/git/show/{path...}?commit=hash.
func (h *FileAPIHandler) ShowVersion(w http.ResponseWriter, r *http.Request) {
	username, _ := auth.UsernameFromContext(r.Context())
	relPath := r.PathValue("path")
	commit := r.URL.Query().Get("commit")

	if !h.store.VCS().Enabled() {
		response.Error(w, http.StatusServiceUnavailable, "git versioning is disabled")
		return
	}
	if commit == "" {
		response.Error(w, http.StatusBadRequest, "commit query parameter is required")
		return
	}

	content, err := h.store.VCS().Show(username, relPath, commit)
	if err != nil {
		response.Error(w, http.StatusNotFound, "revision not found")
		return
	}

	if r.URL.Query().Get("meta") == "1" {
		response.JSON(w, http.StatusOK, map[string]any{
			"path":    relPath,
			"commit":  commit,
			"content": string(content),
		})
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
}

type restoreRequest struct {
	Commit string `json:"commit"`
}

// Restore handles POST /api/git/restore/{path...}.
func (h *FileAPIHandler) Restore(w http.ResponseWriter, r *http.Request) {
	username, _ := auth.UsernameFromContext(r.Context())
	relPath := r.PathValue("path")

	if !h.store.VCS().Enabled() {
		response.Error(w, http.StatusServiceUnavailable, "git versioning is disabled")
		return
	}

	var req restoreRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Commit == "" {
		response.Error(w, http.StatusBadRequest, "commit is required")
		return
	}

	rec, err := h.store.RestoreVersion(username, relPath, req.Commit)
	if err != nil {
		slog.Error("restore failed", "user", username, "path", relPath, "commit", req.Commit, "error", err)
		response.Error(w, http.StatusInternalServerError, "restore failed")
		return
	}

	slog.Info("file restored", "user", username, "path", relPath, "commit", req.Commit)
	response.JSON(w, http.StatusOK, map[string]any{
		"message": "restored",
		"file":    toFileResponse(h.cfg, *rec),
	})
}