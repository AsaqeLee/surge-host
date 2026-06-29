package vcs

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/asaqe/surge-host/internal/config"
)

// CommitInfo describes a single Git revision.
type CommitInfo struct {
	Hash      string    `json:"hash"`
	ShortHash string    `json:"short_hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
}

// Manager manages per-user bare Git repositories with external worktrees.
type Manager struct {
	cfg *config.Config
}

// New creates a Git version control manager.
func New(cfg *config.Config) *Manager {
	return &Manager{cfg: cfg}
}

// Enabled reports whether Git versioning is active.
func (m *Manager) Enabled() bool {
	return m.cfg.GitEnabled
}

// ReposDir returns the directory holding bare repositories.
func (m *Manager) ReposDir() string {
	return filepath.Join(m.cfg.DataDir, "repos")
}

func (m *Manager) gitDir(username string) string {
	return filepath.Join(m.ReposDir(), username+".git")
}

func (m *Manager) workTree(username string) string {
	return m.cfg.UserDir(username)
}

// EnsureRepo initializes a bare repository for the user if needed.
func (m *Manager) EnsureRepo(username string) error {
	if !m.Enabled() {
		return nil
	}

	gitDir := m.gitDir(username)
	if _, err := os.Stat(filepath.Join(gitDir, "HEAD")); err == nil {
		return nil
	}

	if err := os.MkdirAll(m.ReposDir(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(m.workTree(username), 0o755); err != nil {
		return err
	}

	if _, err := m.runBare("init", "--bare", gitDir); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	slog.Info("git repo initialized", "user", username, "repo", gitDir)
	return nil
}

// InitialImport creates the first commit when the repo has no history.
func (m *Manager) InitialImport(username string) error {
	if !m.Enabled() {
		return nil
	}
	if err := m.EnsureRepo(username); err != nil {
		return err
	}
	if m.HasCommits(username) {
		return nil
	}
	return m.CommitAll(username, "initial import")
}

// HasCommits returns true when the repository has at least one commit.
func (m *Manager) HasCommits(username string) bool {
	if !m.Enabled() {
		return false
	}
	_, err := m.run(username, "rev-parse", "HEAD")
	return err == nil
}

// CommitAll stages all changes and commits with the given message.
func (m *Manager) CommitAll(username, message string) error {
	if !m.Enabled() {
		return nil
	}
	if err := m.EnsureRepo(username); err != nil {
		return err
	}
	if _, err := m.run(username, "add", "-A"); err != nil {
		return err
	}
	return m.commitIfChanges(username, message)
}

// Commit stages specific paths and commits with the given message.
func (m *Manager) Commit(username, message string, paths ...string) error {
	if !m.Enabled() {
		return nil
	}
	if err := m.EnsureRepo(username); err != nil {
		return err
	}
	if len(paths) == 0 {
		return m.CommitAll(username, message)
	}
	args := append([]string{"add", "--"}, paths...)
	if _, err := m.run(username, args...); err != nil {
		return err
	}
	return m.commitIfChanges(username, message)
}

// Log returns commit history for a file (newest first).
func (m *Manager) Log(username, relPath string, limit int) ([]CommitInfo, error) {
	if !m.Enabled() {
		return nil, fmt.Errorf("git versioning is disabled")
	}
	if !m.HasCommits(username) {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	format := "%H%x00%h%x00%an%x00%at%x00%s"
	out, err := m.run(username, "log", "--follow",
		fmt.Sprintf("-n%d", limit),
		fmt.Sprintf("--pretty=format:%s", format),
		"--", relPath,
	)
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var commits []CommitInfo
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 5 {
			continue
		}
		ts, _ := strconv.ParseInt(parts[3], 10, 64)
		commits = append(commits, CommitInfo{
			Hash:      parts[0],
			ShortHash: parts[1],
			Author:    parts[2],
			Timestamp: time.Unix(ts, 0).UTC(),
			Message:   parts[4],
		})
	}
	return commits, nil
}

// Show returns file content at a specific commit.
func (m *Manager) Show(username, relPath, commit string) ([]byte, error) {
	if !m.Enabled() {
		return nil, fmt.Errorf("git versioning is disabled")
	}
	rev := fmt.Sprintf("%s:%s", commit, relPath)
	out, err := m.run(username, "show", rev)
	if err != nil {
		return nil, fmt.Errorf("revision not found")
	}
	return []byte(out), nil
}

// Restore checks out a file at a specific commit into the worktree.
func (m *Manager) Restore(username, relPath, commit string) error {
	if !m.Enabled() {
		return fmt.Errorf("git versioning is disabled")
	}
	if _, err := m.run(username, "checkout", commit, "--", relPath); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	msg := fmt.Sprintf("restore: %s to %s", relPath, shortHash(commit))
	return m.Commit(username, msg, relPath)
}

func (m *Manager) commitIfChanges(username, message string) error {
	status, err := m.run(username, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) == "" {
		return nil
	}
	_, err = m.run(username, "commit", "-m", message)
	return err
}

func (m *Manager) run(username string, args ...string) (string, error) {
	gitDir, err := filepath.Abs(m.gitDir(username))
	if err != nil {
		return "", err
	}
	workTree, err := filepath.Abs(m.workTree(username))
	if err != nil {
		return "", err
	}
	return m.exec(args, map[string]string{
		"GIT_DIR":       gitDir,
		"GIT_WORK_TREE": workTree,
	})
}

func (m *Manager) runBare(args ...string) (string, error) {
	return m.exec(args, nil)
}

func (m *Manager) exec(args []string, env map[string]string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+m.cfg.GitAuthorName,
		"GIT_AUTHOR_EMAIL="+m.cfg.GitAuthorEmail,
		"GIT_COMMITTER_NAME="+m.cfg.GitAuthorName,
		"GIT_COMMITTER_EMAIL="+m.cfg.GitAuthorEmail,
	)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

func shortHash(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}