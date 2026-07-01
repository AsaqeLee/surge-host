package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/asaqe/surge-host/internal/auth"
	"github.com/asaqe/surge-host/internal/config"
	"github.com/asaqe/surge-host/internal/db"
	"github.com/asaqe/surge-host/internal/handler"
	"github.com/asaqe/surge-host/internal/middleware"
	"github.com/asaqe/surge-host/internal/store"
	"github.com/asaqe/surge-host/internal/vcs"
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	cfg  *config.Config
	http *http.Server
	db   *db.DB
}

// New creates and configures the HTTP server.
func New(cfg *config.Config) (*Server, error) {
	if err := ensureDirs(cfg); err != nil {
		return nil, err
	}

	database, err := db.Open(cfg.DBPath())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	gitMgr := vcs.New(cfg)
	fileStore, err := store.New(cfg, database, gitMgr)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("init file store: %w", err)
	}

	authSvc := auth.NewService(cfg.JWTSecret, cfg.AdminUser, cfg.AdminPassword)
	if !authSvc.AuthEnabled() {
		slog.Warn("authentication disabled — set SURGE_HOST_ADMIN_PASSWORD to enable")
	} else {
		warnInsecureAuth(cfg)
	}
	if cfg.JWTSecret == "change-me-in-production" {
		slog.Warn("using default JWT secret — set SURGE_HOST_JWT_SECRET to a random value")
	}

	home, err := handler.NewHomeHandler(cfg, authSvc, fileStore)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("init home handler: %w", err)
	}

	pages, err := handler.NewPageHandler(cfg, authSvc)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("init page handler: %w", err)
	}

	raw := handler.NewRawHandler(cfg)
	health := &handler.HealthHandler{}
	authHandler := handler.NewAuthHandler(authSvc)
	fileAPI := handler.NewFileAPIHandler(cfg, fileStore)
	requireAuth := auth.RequireAuth(authSvc)

	mux := http.NewServeMux()

	// Pages
	mux.Handle("GET /{$}", home)
	mux.Handle("GET /healthz", health)

	// Raw URL — core subscription endpoint for Surge
	mux.Handle("GET /raw/{user}/{path...}", raw)
	mux.Handle("HEAD /raw/{user}/{path...}", raw)

	// Static assets
	staticFS := http.FileServer(http.Dir(cfg.StaticDir))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticFS))

	// Auth API
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	// Git version control API (protected)
	mux.Handle("GET /api/git/log/{path...}", requireAuth(http.HandlerFunc(fileAPI.History)))
	mux.Handle("GET /api/git/show/{path...}", requireAuth(http.HandlerFunc(fileAPI.ShowVersion)))
	mux.Handle("POST /api/git/restore/{path...}", requireAuth(http.HandlerFunc(fileAPI.Restore)))

	// Validation API (protected)
	mux.Handle("POST /api/validate", requireAuth(http.HandlerFunc(fileAPI.Validate)))

	// File management API (protected)
	mux.Handle("GET /api/files", requireAuth(http.HandlerFunc(fileAPI.List)))
	mux.Handle("POST /api/files", requireAuth(http.HandlerFunc(fileAPI.Upload)))
	mux.Handle("GET /api/files/{path...}", requireAuth(http.HandlerFunc(fileAPI.Get)))
	mux.Handle("PUT /api/files/{path...}", requireAuth(http.HandlerFunc(fileAPI.Update)))
	mux.Handle("DELETE /api/files/{path...}", requireAuth(http.HandlerFunc(fileAPI.Delete)))
	mux.Handle("PATCH /api/files/{path...}", requireAuth(http.HandlerFunc(fileAPI.Rename)))

	// Management pages
	mux.HandleFunc("GET /upload", pages.Upload)
	mux.HandleFunc("GET /files", pages.Files)
	mux.HandleFunc("GET /edit/{path...}", pages.Edit)

	h := middleware.Recovery(
		middleware.Logging(
			middleware.CORS(mux),
		),
	)

	addr := fmt.Sprintf(":%d", cfg.Port)
	s := &Server{
		cfg: cfg,
		db:  database,
		http: &http.Server{
			Addr:    addr,
			Handler: h,
		},
	}

	return s, nil
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	slog.Info("server starting",
		"addr", s.http.Addr,
		"domain", s.cfg.Domain,
		"data_dir", s.cfg.DataDir,
		"db", s.cfg.DBPath(),
	)
	return s.http.ListenAndServe()
}

// Close releases server resources.
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func warnInsecureAuth(cfg *config.Config) {
	if len(cfg.AdminPassword) < 12 {
		slog.Warn("admin password is short — use at least 12 characters")
	}
	if cfg.AdminPassword == cfg.AdminUser {
		slog.Warn("admin password equals username — use a distinct strong password")
	}
}

func ensureDirs(cfg *config.Config) error {
	dirs := []string{
		cfg.DataDir,
		cfg.UsersDir(),
		cfg.UserDir(cfg.AdminUser),
	}
	if cfg.GitEnabled {
		dirs = append(dirs, filepath.Join(cfg.DataDir, "repos"))
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	if _, err := os.Stat(cfg.StaticDir); os.IsNotExist(err) {
		return fmt.Errorf("static directory not found: %s", cfg.StaticDir)
	}
	if _, err := os.Stat(cfg.TemplatesDir); os.IsNotExist(err) {
		return fmt.Errorf("templates directory not found: %s", cfg.TemplatesDir)
	}
	return nil
}

