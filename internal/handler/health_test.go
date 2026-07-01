package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asaqe/surge-host/internal/config"
)

type fakeHealthPinger struct {
	err error
}

func (f fakeHealthPinger) Ping() error {
	return f.err
}

func TestHealthHandlerOK(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		DataDir:    dataDir,
		GitEnabled: true,
	}
	if err := ensureTestDir(filepath.Join(dataDir, "users")); err != nil {
		t.Fatalf("prepare users dir: %v", err)
	}
	if err := ensureTestDir(filepath.Join(dataDir, "repos")); err != nil {
		t.Fatalf("prepare repos dir: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	NewHealthHandler(cfg, fakeHealthPinger{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected ok response, got %s", rr.Body.String())
	}
}

func TestHealthHandlerReturns503WhenDatabaseFails(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		DataDir:    dataDir,
		GitEnabled: false,
	}
	if err := ensureTestDir(filepath.Join(dataDir, "users")); err != nil {
		t.Fatalf("prepare users dir: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	NewHealthHandler(cfg, fakeHealthPinger{err: errors.New("db down")}).ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"database":"error: db down"`) {
		t.Fatalf("expected database failure in response, got %s", rr.Body.String())
	}
}

func TestHealthHandlerReturns503WhenReposDirMissing(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		DataDir:    dataDir,
		GitEnabled: true,
	}
	if err := ensureTestDir(filepath.Join(dataDir, "users")); err != nil {
		t.Fatalf("prepare users dir: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	NewHealthHandler(cfg, fakeHealthPinger{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"repos_dir":"error:`) {
		t.Fatalf("expected repos_dir failure in response, got %s", rr.Body.String())
	}
}

func ensureTestDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}
