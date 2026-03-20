package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS verification_tokens (
    id TEXT PRIMARY KEY,
    pocket_id_user_id TEXT NOT NULL UNIQUE,
    rsi_handle TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    expires_at DATETIME NOT NULL
);
`

// VerificationToken represents a row in the verification_tokens table.
type VerificationToken struct {
	ID             string
	PocketIDUserID string
	RSIHandle      string
	Token          string
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

// Store provides access to the SQLite database.
type Store struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database at path and runs migrations.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// UpsertToken inserts a new verification token for the user, or returns the
// existing one if the same handle is already pending for this user.
// It replaces any existing token for a different handle.
func (s *Store) UpsertToken(ctx context.Context, vt *VerificationToken) (*VerificationToken, error) {
	existing, err := s.GetTokenByUserID(ctx, vt.PocketIDUserID)
	if err != nil && err != ErrNotFound {
		return nil, err
	}
	if existing != nil && existing.RSIHandle == vt.RSIHandle && existing.ExpiresAt.After(time.Now()) {
		return existing, nil
	}

	if err := s.DeleteTokenByUserID(ctx, vt.PocketIDUserID); err != nil {
		return nil, err
	}

	const q = `
INSERT INTO verification_tokens (id, pocket_id_user_id, rsi_handle, token, expires_at)
VALUES (?, ?, ?, ?, ?)`
	_, err = s.db.ExecContext(ctx, q,
		vt.ID, vt.PocketIDUserID, vt.RSIHandle, vt.Token,
		vt.ExpiresAt.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("insert token: %w", err)
	}
	return vt, nil
}

// GetTokenByUserID retrieves the pending verification token for a user.
// Returns ErrNotFound if no token exists.
func (s *Store) GetTokenByUserID(ctx context.Context, userID string) (*VerificationToken, error) {
	const q = `
SELECT id, pocket_id_user_id, rsi_handle, token, created_at, expires_at
FROM verification_tokens
WHERE pocket_id_user_id = ?`
	row := s.db.QueryRowContext(ctx, q, userID)
	return scanToken(row)
}

// HandleIsLinkedToOtherUser reports whether the given RSI handle is already
// linked to a different SCID user than excludeUserID.
func (s *Store) HandleIsLinkedToOtherUser(ctx context.Context, handle, excludeUserID string) (bool, error) {
	const q = `
SELECT COUNT(*) FROM verification_tokens
WHERE rsi_handle = ? AND pocket_id_user_id != ?`
	var count int
	if err := s.db.QueryRowContext(ctx, q, handle, excludeUserID).Scan(&count); err != nil {
		return false, fmt.Errorf("check handle uniqueness: %w", err)
	}
	return count > 0, nil
}

// DeleteTokenByUserID removes any pending verification token for the user.
func (s *Store) DeleteTokenByUserID(ctx context.Context, userID string) error {
	const q = `DELETE FROM verification_tokens WHERE pocket_id_user_id = ?`
	if _, err := s.db.ExecContext(ctx, q, userID); err != nil {
		return fmt.Errorf("delete token: %w", err)
	}
	return nil
}

// scanToken scans a single VerificationToken from a *sql.Row.
func scanToken(row *sql.Row) (*VerificationToken, error) {
	var vt VerificationToken
	var createdAt, expiresAt string
	err := row.Scan(&vt.ID, &vt.PocketIDUserID, &vt.RSIHandle, &vt.Token, &createdAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan token: %w", err)
	}
	vt.CreatedAt, _ = parseTime(createdAt)
	vt.ExpiresAt, _ = parseTime(expiresAt)
	return &vt, nil
}

// parseTime parses a time string stored in SQLite, trying RFC3339 first (new
// rows) and falling back to the SQLite datetime() format (legacy rows).
func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse(time.DateTime, s)
}

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = fmt.Errorf("not found")
