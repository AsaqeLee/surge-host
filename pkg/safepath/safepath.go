package safepath

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NormalizeUserPath strips a single leading "{username}/" prefix if present.
// Only the first path segment is considered; interior segments are untouched.
func NormalizeUserPath(username, rel string) string {
	if username == "" || rel == "" {
		return rel
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	prefix := username + "/"
	if strings.HasPrefix(rel, prefix) {
		return rel[len(prefix):]
	}
	return rel
}

// PrepareUserPath trims, cleans, strips a leading username prefix, and validates.
func PrepareUserPath(username, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	rel = NormalizeUserPath(username, rel)
	if err := Validate(rel); err != nil {
		return "", err
	}
	return rel, nil
}

// Validate ensures the relative path is safe (no traversal, no absolute paths).
func Validate(rel string) error {
	if rel == "" {
		return fmt.Errorf("empty path")
	}
	if strings.Contains(rel, "\x00") {
		return fmt.Errorf("invalid path")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." {
		return fmt.Errorf("invalid path")
	}
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, string(filepath.Separator)+"..") {
		return fmt.Errorf("path traversal detected")
	}
	return nil
}

// JoinSafe joins base and relative paths after validating the relative segment.
func JoinSafe(base, rel string) (string, error) {
	if err := Validate(rel); err != nil {
		return "", err
	}
	full := filepath.Join(base, rel)
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve base path: %w", err)
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("resolve full path: %w", err)
	}
	relToBase, err := filepath.Rel(absBase, absFull)
	if err != nil {
		return "", fmt.Errorf("path outside base: %w", err)
	}
	if relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside base directory")
	}
	return full, nil
}

// AllowedExtension checks if the filename has an allowed extension.
func AllowedExtension(filename string, allowed map[string]bool) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return allowed[ext]
}