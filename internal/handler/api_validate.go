package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/asaqe/surge-host/internal/auth"
	"github.com/asaqe/surge-host/internal/store"
	"github.com/asaqe/surge-host/pkg/response"
	"github.com/asaqe/surge-host/pkg/safepath"
	"github.com/asaqe/surge-host/pkg/validator"
)

type validateRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Validate handles POST /api/validate — check Surge syntax without saving.
func (h *FileAPIHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, h.cfg.MaxFileSize+4096)).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	username, _ := auth.UsernameFromContext(r.Context())
	path, err := safepath.PrepareUserPath(username, strings.TrimSpace(req.Path))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid path")
		return
	}

	var content []byte
	if req.Content != "" {
		content = []byte(req.Content)
	} else {
		data, err := h.store.ReadContent(username, path)
		if err != nil {
			if store.IsNotFound(err) {
				response.Error(w, http.StatusNotFound, "file not found")
				return
			}
			response.Error(w, http.StatusInternalServerError, "failed to read file")
			return
		}
		content = data
	}

	if int64(len(content)) > h.cfg.MaxFileSize {
		response.Error(w, http.StatusRequestEntityTooLarge, "content too large")
		return
	}

	result := h.store.ValidateContent(path, content)
	response.JSON(w, http.StatusOK, result)
}

// validationErrorResponse writes validation issues as JSON 422.
func validationErrorResponse(w http.ResponseWriter, ve *validator.ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":      "validation failed",
		"validation": ve.Result,
	})
}