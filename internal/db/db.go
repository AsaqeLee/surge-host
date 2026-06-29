package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection.
type DB struct {
	*sql.DB
}

// Open opens (or creates) the SQLite database and runs migrations.
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	d := &DB{DB: sqlDB}
	if err := d.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) migrate() error {
	_, err := d.Exec(schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// EnsureUser creates the user if it does not exist and returns the record.
func (d *DB) EnsureUser(username string) (*User, error) {
	var u User
	err := d.QueryRow(
		`SELECT id, username, created_at FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.CreatedAt)
	if err == nil {
		return &u, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	res, err := d.Exec(`INSERT INTO users (username) VALUES (?)`, username)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	u = User{ID: id, Username: username, CreatedAt: time.Now()}
	return &u, nil
}

// UpsertFile inserts or updates file metadata.
func (d *DB) UpsertFile(userID int64, path string, size int64, isPublic bool) (*FileRecord, error) {
	pub := 0
	if isPublic {
		pub = 1
	}
	_, err := d.Exec(`
		INSERT INTO files (user_id, path, size, is_public, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(user_id, path) DO UPDATE SET
			size = excluded.size,
			is_public = excluded.is_public,
			updated_at = datetime('now')
	`, userID, path, size, pub)
	if err != nil {
		return nil, err
	}
	return d.GetFileByUserIDAndPath(userID, path)
}

// GetFileByUserIDAndPath returns a file record by user ID and relative path.
func (d *DB) GetFileByUserIDAndPath(userID int64, path string) (*FileRecord, error) {
	var f FileRecord
	var pub int
	err := d.QueryRow(`
		SELECT f.id, f.user_id, u.username, f.path, f.size, f.is_public, f.created_at, f.updated_at
		FROM files f JOIN users u ON f.user_id = u.id
		WHERE f.user_id = ? AND f.path = ?
	`, userID, path).Scan(
		&f.ID, &f.UserID, &f.Username, &f.Path, &f.Size, &pub, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	f.IsPublic = pub == 1
	return &f, nil
}

// GetFileByUsernameAndPath returns a file record by username and relative path.
func (d *DB) GetFileByUsernameAndPath(username, path string) (*FileRecord, error) {
	var f FileRecord
	var pub int
	err := d.QueryRow(`
		SELECT f.id, f.user_id, u.username, f.path, f.size, f.is_public, f.created_at, f.updated_at
		FROM files f JOIN users u ON f.user_id = u.id
		WHERE u.username = ? AND f.path = ?
	`, username, path).Scan(
		&f.ID, &f.UserID, &f.Username, &f.Path, &f.Size, &pub, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	f.IsPublic = pub == 1
	return &f, nil
}

// ListPublicFiles returns all publicly visible files.
func (d *DB) ListPublicFiles() ([]FileRecord, error) {
	rows, err := d.Query(`
		SELECT f.id, f.user_id, u.username, f.path, f.size, f.is_public, f.created_at, f.updated_at
		FROM files f JOIN users u ON f.user_id = u.id
		WHERE f.is_public = 1
		ORDER BY u.username, f.path
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFiles(rows)
}

// ListUserFiles returns all files belonging to a user.
func (d *DB) ListUserFiles(username string) ([]FileRecord, error) {
	rows, err := d.Query(`
		SELECT f.id, f.user_id, u.username, f.path, f.size, f.is_public, f.created_at, f.updated_at
		FROM files f JOIN users u ON f.user_id = u.id
		WHERE u.username = ?
		ORDER BY f.path
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFiles(rows)
}

// DeleteFile removes a file record.
func (d *DB) DeleteFile(userID int64, path string) error {
	_, err := d.Exec(`DELETE FROM files WHERE user_id = ? AND path = ?`, userID, path)
	return err
}

// RenameFile updates the path of a file record.
func (d *DB) RenameFile(userID int64, oldPath, newPath string) error {
	_, err := d.Exec(`
		UPDATE files SET path = ?, updated_at = datetime('now')
		WHERE user_id = ? AND path = ?
	`, newPath, userID, oldPath)
	return err
}

// ListFilePathsByUser returns all tracked paths for a user.
func (d *DB) ListFilePathsByUser(userID int64) (map[string]bool, error) {
	rows, err := d.Query(`SELECT path FROM files WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]bool)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		m[p] = true
	}
	return m, rows.Err()
}

func scanFiles(rows *sql.Rows) ([]FileRecord, error) {
	var files []FileRecord
	for rows.Next() {
		var f FileRecord
		var pub int
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.Username, &f.Path, &f.Size, &pub, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		f.IsPublic = pub == 1
		files = append(files, f)
	}
	return files, rows.Err()
}