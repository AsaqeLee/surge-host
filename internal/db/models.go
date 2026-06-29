package db

import "time"

// User represents a registered user (single-user mode starts with admin).
type User struct {
	ID        int64
	Username  string
	CreatedAt time.Time
}

// FileRecord stores metadata for a user-hosted Surge config file.
type FileRecord struct {
	ID        int64
	UserID    int64
	Username  string
	Path      string // relative path within user dir, forward slashes
	Size      int64
	IsPublic  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}