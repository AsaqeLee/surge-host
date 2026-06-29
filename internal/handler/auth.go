package handler

import (
	"encoding/json"
	"net/http"

	"github.com/asaqe/surge-host/internal/auth"
	"github.com/asaqe/surge-host/pkg/response"
)

// AuthHandler handles login requests.
type AuthHandler struct {
	svc *auth.Service
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Expires  string `json:"expires_in"`
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !h.svc.AuthEnabled() {
		response.Error(w, http.StatusBadRequest, "authentication is disabled (set SURGE_HOST_ADMIN_PASSWORD)")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	token, err := h.svc.Authenticate(req.Username, req.Password)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	response.JSON(w, http.StatusOK, loginResponse{
		Token:    token,
		Username: req.Username,
		Expires:  "24h",
	})
}