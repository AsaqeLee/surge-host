package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/asaqe/surge-host/internal/auth"
	"github.com/asaqe/surge-host/internal/config"
	"github.com/asaqe/surge-host/internal/db"
	"github.com/asaqe/surge-host/internal/store"
	"github.com/asaqe/surge-host/pkg/response"
	"github.com/asaqe/surge-host/pkg/safepath"
	"github.com/asaqe/surge-host/pkg/validator"
)

// FileAPIHandler handles REST endpoints for file management.
type FileAPIHandler struct {
	cfg   *config.Config
	store *store.FileStore
}

// NewFileAPIHandler creates a FileAPIHandler.
func NewFileAPIHandler(cfg *config.Config, fs *store.FileStore) *FileAPIHandler {
	return &FileAPIHandler{cfg: cfg, store: fs}
}

// fileResponse is the JSON representation of a file record.
type fileResponse struct {
	ID        int64  `json:"id"`
	User      string `json:"user"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	IsPublic  bool   `json:"is_public"`
	RawURL    string `json:"raw_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toFileResponse(cfg *config.Config, f db.FileRecord) fileResponse {
	return fileResponse{
		ID:        f.ID,
		User:      f.Username,
		Path:      f.Path,
		Size:      f.Size,
		IsPublic:  f.IsPublic,
		RawURL:    cfg.RawURL(f.Username, f.Path),
		CreatedAt: f.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: f.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// List handles GET /api/files.
func (h *FileAPIHandler) List(w http.ResponseWriter, r *http.Request) {
	var files []db.FileRecord
	var err error

	if username, ok := auth.UsernameFromContext(r.Context()); ok {
		files, err = h.store.ListByUser(username)
	} else {
		files, err = h.store.ListPublic()
	}
	if err != nil {
		slog.Error("list files failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to list files")
		return
	}

	result := make([]fileResponse, 0, len(files))
	for _, f := range files {
		result = append(result, toFileResponse(h.cfg, f))
	}
	response.JSON(w, http.StatusOK, map[string]any{"files": result})
}

// Get handles GET /api/files/{path...}.
func (h *FileAPIHandler) Get(w http.ResponseWriter, r *http.Request) {
	username, _ := auth.UsernameFromContext(r.Context())
	relPath := r.PathValue("path")

	rec, err := h.store.Get(username, relPath)
	if err != nil {
		if store.IsNotFound(err) {
			response.Error(w, http.StatusNotFound, "file not found")
			return
		}
		slog.Error("get file metadata failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to get file")
		return
	}

	if r.Header.Get("Accept") == "application/json" || r.URL.Query().Get("meta") == "1" {
		resp := toFileResponse(h.cfg, *rec)
		if r.URL.Query().Get("content") == "1" {
			content, err := h.store.ReadContent(username, relPath)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "failed to read file")
				return
			}
			response.JSON(w, http.StatusOK, map[string]any{
				"file":    resp,
				"content": string(content),
			})
			return
		}
		response.JSON(w, http.StatusOK, resp)
		return
	}

	content, err := h.store.ReadContent(username, relPath)
	if err != nil {
		if store.IsNotFound(err) {
			response.Error(w, http.StatusNotFound, "file not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
}

// Upload handles POST /api/files.
func (h *FileAPIHandler) Upload(w http.ResponseWriter, r *http.Request) {
	username, _ := auth.UsernameFromContext(r.Context())

	if err := r.ParseMultipartForm(h.cfg.MaxFileSize); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid multipart form or file too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	relPath := strings.TrimSpace(r.FormValue("path"))
	if relPath == "" {
		relPath = header.Filename
	}
	relPath, err = safepath.PrepareUserPath(username, relPath)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid path")
		return
	}

	rec, err := h.store.Save(username, relPath, file, header.Size)
	if err != nil {
		if handleSaveError(w, err) {
			return
		}
		slog.Error("upload failed", "error", err, "path", relPath)
		response.Error(w, http.StatusInternalServerError, "upload failed")
		return
	}

	slog.Info("file uploaded", "user", username, "path", relPath, "size", rec.Size)
	response.JSON(w, http.StatusCreated, toFileResponse(h.cfg, *rec))
}

// Update handles PUT /api/files/{path...} — replace file content.
func (h *FileAPIHandler) Update(w http.ResponseWriter, r *http.Request) {
	username, _ := auth.UsernameFromContext(r.Context())
	relPath := r.PathValue("path")

	if err := safepath.Validate(relPath); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid path")
		return
	}

	body := http.MaxBytesReader(w, r.Body, h.cfg.MaxFileSize+1)
	defer body.Close()

	rec, err := h.store.Save(username, relPath, body, h.cfg.MaxFileSize)
	if err != nil {
		if handleSaveError(w, err) {
			return
		}
		slog.Error("update failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "update failed")
		return
	}

	slog.Info("file updated", "user", username, "path", relPath)
	response.JSON(w, http.StatusOK, toFileResponse(h.cfg, *rec))
}

// Delete handles DELETE /api/files/{path...}.
func (h *FileAPIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	username, _ := auth.UsernameFromContext(r.Context())
	relPath := r.PathValue("path")

	if err := h.store.Delete(username, relPath); err != nil {
		if store.IsNotFound(err) {
			response.Error(w, http.StatusNotFound, "file not found")
			return
		}
		slog.Error("delete failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "delete failed")
		return
	}

	slog.Info("file deleted", "user", username, "path", relPath)
	response.JSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

type renameRequest struct {
	NewPath string `json:"new_path"`
}

// Rename handles PATCH /api/files/{path...}.
func (h *FileAPIHandler) Rename(w http.ResponseWriter, r *http.Request) {
	username, _ := auth.UsernameFromContext(r.Context())
	oldPath := r.PathValue("path")

	var req renameRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	newPath, err := safepath.PrepareUserPath(username, req.NewPath)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid new_path")
		return
	}

	rec, err := h.store.Rename(username, oldPath, newPath)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "not allowed") {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		slog.Error("rename failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "rename failed")
		return
	}

	slog.Info("file renamed", "user", username, "from", oldPath, "to", newPath)
	response.JSON(w, http.StatusOK, toFileResponse(h.cfg, *rec))
}

func handleSaveError(w http.ResponseWriter, err error) bool {
	var ve *validator.ValidationError
	if errors.As(err, &ve) {
		validationErrorResponse(w, ve)
		return true
	}
	if strings.Contains(err.Error(), "not allowed") {
		response.Error(w, http.StatusForbidden, err.Error())
		return true
	}
	if strings.Contains(err.Error(), "size limit") {
		response.Error(w, http.StatusRequestEntityTooLarge, err.Error())
		return true
	}
	return false
}