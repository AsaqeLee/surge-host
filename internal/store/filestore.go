package store

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/asaqe/surge-host/internal/config"
	"github.com/asaqe/surge-host/internal/db"
	"github.com/asaqe/surge-host/internal/vcs"
	"github.com/asaqe/surge-host/pkg/safepath"
	"github.com/asaqe/surge-host/pkg/validator"
)

// FileStore manages file metadata in SQLite and content on disk.
type FileStore struct {
	cfg *config.Config
	db  *db.DB
	vcs *vcs.Manager
}

// New creates a FileStore and syncs existing files from disk.
func New(cfg *config.Config, database *db.DB, git *vcs.Manager) (*FileStore, error) {
	s := &FileStore{cfg: cfg, db: database, vcs: git}
	if err := s.SyncFromDisk(cfg.AdminUser); err != nil {
		return nil, fmt.Errorf("sync files: %w", err)
	}
	if err := git.InitialImport(cfg.AdminUser); err != nil {
		return nil, fmt.Errorf("git initial import: %w", err)
	}
	return s, nil
}

// VCS returns the version control manager.
func (s *FileStore) VCS() *vcs.Manager {
	return s.vcs
}

// SyncFromDisk imports filesystem files into the database for a user.
func (s *FileStore) SyncFromDisk(username string) error {
	user, err := s.db.EnsureUser(username)
	if err != nil {
		return err
	}

	userDir := s.cfg.UserDir(username)
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return err
	}

	seen := make(map[string]bool)
	err = filepath.WalkDir(userDir, func(fullPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if !safepath.AllowedExtension(name, s.cfg.AllowedExtensions) {
			return nil
		}
		rel, err := filepath.Rel(userDir, fullPath)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return nil
		}
		seen[rel] = true
		if _, err := s.db.UpsertFile(user.ID, rel, info.Size(), true); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	tracked, err := s.db.ListFilePathsByUser(user.ID)
	if err != nil {
		return err
	}
	for path := range tracked {
		if !seen[path] {
			if err := s.db.DeleteFile(user.ID, path); err != nil {
				slog.Warn("remove orphan db record", "path", path, "error", err)
			}
		}
	}
	return nil
}

// ListPublic returns all public file records.
func (s *FileStore) ListPublic() ([]db.FileRecord, error) {
	return s.db.ListPublicFiles()
}

// ListByUser returns all files for a user.
func (s *FileStore) ListByUser(username string) ([]db.FileRecord, error) {
	return s.db.ListUserFiles(username)
}

// Get returns file metadata.
func (s *FileStore) Get(username, relPath string) (*db.FileRecord, error) {
	return s.db.GetFileByUsernameAndPath(username, relPath)
}

// ReadContent reads file content from disk.
func (s *FileStore) ReadContent(username, relPath string) ([]byte, error) {
	full, err := s.resolvePath(username, relPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(full)
}

// Save writes content to disk and upserts metadata.
func (s *FileStore) Save(username, relPath string, r io.Reader, size int64) (*db.FileRecord, error) {
	if !safepath.AllowedExtension(relPath, s.cfg.AllowedExtensions) {
		return nil, fmt.Errorf("file type not allowed")
	}
	if size > s.cfg.MaxFileSize {
		return nil, fmt.Errorf("file exceeds size limit (%d bytes)", s.cfg.MaxFileSize)
	}

	content, err := io.ReadAll(io.LimitReader(r, s.cfg.MaxFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > s.cfg.MaxFileSize {
		return nil, fmt.Errorf("file exceeds size limit (%d bytes)", s.cfg.MaxFileSize)
	}

	if err := s.validateContent(relPath, content); err != nil {
		return nil, err
	}

	full, err := s.resolvePath(username, relPath)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(full, content, 0o644); err != nil {
		return nil, err
	}
	written := int64(len(content))

	user, err := s.db.EnsureUser(username)
	if err != nil {
		return nil, err
	}
	rec, err := s.db.UpsertFile(user.ID, relPath, written, true)
	if err != nil {
		return nil, err
	}

	if err := s.vcs.Commit(username, fmt.Sprintf("update: %s", relPath), relPath); err != nil {
		slog.Warn("git commit failed", "user", username, "path", relPath, "error", err)
	}
	return rec, nil
}

// Delete removes a file from disk and database.
func (s *FileStore) Delete(username, relPath string) error {
	user, err := s.db.EnsureUser(username)
	if err != nil {
		return err
	}

	full, err := s.resolvePath(username, relPath)
	if err != nil {
		return err
	}

	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := s.db.DeleteFile(user.ID, relPath); err != nil {
		return err
	}

	if err := s.vcs.CommitAll(username, fmt.Sprintf("delete: %s", relPath)); err != nil {
		slog.Warn("git commit failed", "user", username, "path", relPath, "error", err)
	}
	return nil
}

// Rename moves a file on disk and updates the database record.
func (s *FileStore) Rename(username, oldPath, newPath string) (*db.FileRecord, error) {
	if !safepath.AllowedExtension(newPath, s.cfg.AllowedExtensions) {
		return nil, fmt.Errorf("file type not allowed")
	}

	user, err := s.db.EnsureUser(username)
	if err != nil {
		return nil, err
	}

	oldFull, err := s.resolvePath(username, oldPath)
	if err != nil {
		return nil, err
	}
	newFull, err := s.resolvePath(username, newPath)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(oldFull); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found")
		}
		return nil, err
	}
	if _, err := os.Stat(newFull); err == nil {
		return nil, fmt.Errorf("destination already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(newFull), 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(oldFull, newFull); err != nil {
		return nil, err
	}

	info, err := os.Stat(newFull)
	if err != nil {
		return nil, err
	}

	if err := s.db.RenameFile(user.ID, oldPath, newPath); err != nil {
		_ = os.Rename(newFull, oldFull)
		return nil, err
	}
	_, err = s.db.Exec(`UPDATE files SET size = ?, updated_at = datetime('now') WHERE user_id = ? AND path = ?`,
		info.Size(), user.ID, newPath)
	if err != nil {
		return nil, err
	}

	if err := s.vcs.CommitAll(username, fmt.Sprintf("rename: %s -> %s", oldPath, newPath)); err != nil {
		slog.Warn("git commit failed", "user", username, "from", oldPath, "to", newPath, "error", err)
	}

	return s.db.GetFileByUserIDAndPath(user.ID, newPath)
}

// RestoreVersion restores a file to a specific Git commit.
func (s *FileStore) RestoreVersion(username, relPath, commit string) (*db.FileRecord, error) {
	if _, err := s.Get(username, relPath); err != nil && !IsNotFound(err) {
		return nil, err
	}

	if err := s.vcs.Restore(username, relPath, commit); err != nil {
		return nil, err
	}

	full, err := s.resolvePath(username, relPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}

	user, err := s.db.EnsureUser(username)
	if err != nil {
		return nil, err
	}
	return s.db.UpsertFile(user.ID, relPath, info.Size(), true)
}

func (s *FileStore) resolvePath(username, relPath string) (string, error) {
	if strings.Contains(username, "/") || strings.Contains(username, "..") {
		return "", fmt.Errorf("invalid username")
	}
	return safepath.JoinSafe(s.cfg.UserDir(username), relPath)
}

// ValidateContent checks Surge syntax for the given path and content.
func (s *FileStore) ValidateContent(relPath string, content []byte) validator.Result {
	if !s.cfg.ValidateEnabled {
		return validator.Result{Valid: true, FileType: filepath.Ext(relPath)}
	}
	return validator.Validate(relPath, content, s.cfg.ValidateStrict)
}

func (s *FileStore) validateContent(relPath string, content []byte) error {
	result := s.ValidateContent(relPath, content)
	if result.HasErrors() {
		return &validator.ValidationError{Result: result}
	}
	return nil
}

// ErrNotFound indicates the requested file does not exist.
var ErrNotFound = errors.New("file not found")

// IsNotFound checks if an error is a not-found condition.
func IsNotFound(err error) bool {
	if errors.Is(err, ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return true
	}
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return false
}