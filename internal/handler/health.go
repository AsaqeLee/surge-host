package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/asaqe/surge-host/internal/config"
)

type healthPinger interface {
	Ping() error
}

// HealthHandler 返回依赖级健康检查结果。
type HealthHandler struct {
	cfg *config.Config
	db  healthPinger
}

// NewHealthHandler 创建带依赖探针的健康检查处理器。
func NewHealthHandler(cfg *config.Config, db healthPinger) *HealthHandler {
	return &HealthHandler{cfg: cfg, db: db}
}

// ServeHTTP handles GET /healthz.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{}
	statusCode := http.StatusOK

	if err := h.db.Ping(); err != nil {
		checks["database"] = "error: " + err.Error()
		statusCode = http.StatusServiceUnavailable
	} else {
		checks["database"] = "ok"
	}

	if err := checkWritableDir(h.cfg.DataDir); err != nil {
		checks["data_dir"] = "error: " + err.Error()
		statusCode = http.StatusServiceUnavailable
	} else {
		checks["data_dir"] = "ok"
	}

	if err := checkReadableDir(h.cfg.UsersDir()); err != nil {
		checks["users_dir"] = "error: " + err.Error()
		statusCode = http.StatusServiceUnavailable
	} else {
		checks["users_dir"] = "ok"
	}

	if h.cfg.GitEnabled {
		reposDir := filepath.Join(h.cfg.DataDir, "repos")
		if err := checkReadableDir(reposDir); err != nil {
			checks["repos_dir"] = "error: " + err.Error()
			statusCode = http.StatusServiceUnavailable
		} else {
			checks["repos_dir"] = "ok"
		}
	}

	status := "ok"
	if statusCode != http.StatusOK {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": status,
		"checks": checks,
	})
}

func checkReadableDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	return nil
}

func checkWritableDir(dir string) error {
	if err := checkReadableDir(dir); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".healthcheck-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
}
