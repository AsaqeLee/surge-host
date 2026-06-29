package handler

import (
	"encoding/json"
	"net/http"
)

// HealthHandler returns a simple health check response.
type HealthHandler struct{}

// ServeHTTP handles GET /healthz.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}