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

-- org_cache stores RSI org metadata indexed by SID.
-- logo_path is the filesystem path to the cached logo image (may be empty).
CREATE TABLE IF NOT EXISTS org_cache (
    sid TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    logo_path TEXT NOT NULL DEFAULT '',
    fetched_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- user_orgs maps Pocket ID user IDs to their RSI org SIDs.
-- is_main = 1 indicates the user's primary org.
CREATE TABLE IF NOT EXISTS user_orgs (
    pocket_id_user_id TEXT NOT NULL,
    sid TEXT NOT NULL,
    rank_name TEXT NOT NULL DEFAULT '',
    is_main INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (pocket_id_user_id, sid)
);

-- app_registrations links Pocket ID OIDC client IDs to SCID owner users.
-- verified_only = 1 means only users in the "verified" group can use this app.
CREATE TABLE IF NOT EXISTS app_registrations (
    id TEXT PRIMARY KEY,
    oidc_client_id TEXT NOT NULL UNIQUE,
    owner_user_id TEXT NOT NULL,
    verified_only INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
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

// AppRegistration tracks which Pocket ID OIDC client belongs to which SCID user.
type AppRegistration struct {
	ID           string
	OIDCClientID string
	OwnerUserID  string
	VerifiedOnly bool
	CreatedAt    time.Time
}

// CreateAppRegistration inserts a new app registration row.
func (s *Store) CreateAppRegistration(ctx context.Context, reg *AppRegistration) error {
	const q = `
INSERT INTO app_registrations (id, oidc_client_id, owner_user_id, verified_only, created_at)
VALUES (?, ?, ?, ?, ?)`
	verifiedOnly := 0
	if reg.VerifiedOnly {
		verifiedOnly = 1
	}
	_, err := s.db.ExecContext(ctx, q,
		reg.ID, reg.OIDCClientID, reg.OwnerUserID, verifiedOnly,
		reg.CreatedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("insert app registration: %w", err)
	}
	return nil
}

// GetAppRegistrationByClientID retrieves an app registration by Pocket ID client ID.
// Returns ErrNotFound if no matching row exists.
func (s *Store) GetAppRegistrationByClientID(ctx context.Context, oidcClientID string) (*AppRegistration, error) {
	const q = `
SELECT id, oidc_client_id, owner_user_id, verified_only, created_at
FROM app_registrations
WHERE oidc_client_id = ?`
	return scanAppRegistration(s.db.QueryRowContext(ctx, q, oidcClientID))
}

// ListAppRegistrationsByOwner returns all app registrations for a given user.
func (s *Store) ListAppRegistrationsByOwner(ctx context.Context, ownerUserID string) ([]AppRegistration, error) {
	const q = `
SELECT id, oidc_client_id, owner_user_id, verified_only, created_at
FROM app_registrations
WHERE owner_user_id = ?
ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, q, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("query app registrations: %w", err)
	}
	defer rows.Close()

	var result []AppRegistration
	for rows.Next() {
		reg, err := scanAppRegistrationRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *reg)
	}
	return result, rows.Err()
}

// DeleteAppRegistrationByClientID removes an app registration by OIDC client ID.
func (s *Store) DeleteAppRegistrationByClientID(ctx context.Context, oidcClientID string) error {
	const q = `DELETE FROM app_registrations WHERE oidc_client_id = ?`
	if _, err := s.db.ExecContext(ctx, q, oidcClientID); err != nil {
		return fmt.Errorf("delete app registration: %w", err)
	}
	return nil
}

// UpdateAppRegistrationVerifiedOnly updates the verified_only flag for an app.
func (s *Store) UpdateAppRegistrationVerifiedOnly(ctx context.Context, oidcClientID string, verifiedOnly bool) error {
	flag := 0
	if verifiedOnly {
		flag = 1
	}
	const q = `UPDATE app_registrations SET verified_only = ? WHERE oidc_client_id = ?`
	if _, err := s.db.ExecContext(ctx, q, flag, oidcClientID); err != nil {
		return fmt.Errorf("update app registration: %w", err)
	}
	return nil
}

func scanAppRegistration(row *sql.Row) (*AppRegistration, error) {
	var reg AppRegistration
	var verifiedOnly int
	var createdAt string
	err := row.Scan(&reg.ID, &reg.OIDCClientID, &reg.OwnerUserID, &verifiedOnly, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan app registration: %w", err)
	}
	reg.VerifiedOnly = verifiedOnly == 1
	reg.CreatedAt, _ = parseTime(createdAt)
	return &reg, nil
}

func scanAppRegistrationRow(rows *sql.Rows) (*AppRegistration, error) {
	var reg AppRegistration
	var verifiedOnly int
	var createdAt string
	err := rows.Scan(&reg.ID, &reg.OIDCClientID, &reg.OwnerUserID, &verifiedOnly, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan app registration: %w", err)
	}
	reg.VerifiedOnly = verifiedOnly == 1
	reg.CreatedAt, _ = parseTime(createdAt)
	return &reg, nil
}

// OrgCacheEntry represents a row in the org_cache table.
type OrgCacheEntry struct {
	SID       string
	Name      string
	LogoPath  string
	FetchedAt time.Time
}

// UserOrg represents a user's membership in an RSI org.
type UserOrg struct {
	PocketIDUserID string
	SID            string
	RankName       string
	IsMain         bool
}

// UpsertOrgCache inserts or replaces an org cache entry.
func (s *Store) UpsertOrgCache(ctx context.Context, e *OrgCacheEntry) error {
	const q = `
INSERT INTO org_cache (sid, name, logo_path, fetched_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(sid) DO UPDATE SET
    name = excluded.name,
    logo_path = excluded.logo_path,
    fetched_at = excluded.fetched_at`
	_, err := s.db.ExecContext(ctx, q,
		e.SID, e.Name, e.LogoPath, e.FetchedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert org cache: %w", err)
	}
	return nil
}

// GetOrgCache retrieves a cached org entry by SID. Returns ErrNotFound if missing.
func (s *Store) GetOrgCache(ctx context.Context, sid string) (*OrgCacheEntry, error) {
	const q = `SELECT sid, name, logo_path, fetched_at FROM org_cache WHERE sid = ?`
	var e OrgCacheEntry
	var fetchedAt string
	err := s.db.QueryRowContext(ctx, q, sid).Scan(&e.SID, &e.Name, &e.LogoPath, &fetchedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get org cache: %w", err)
	}
	e.FetchedAt, _ = parseTime(fetchedAt)
	return &e, nil
}

// SetUserOrgs replaces all org memberships for a user.
func (s *Store) SetUserOrgs(ctx context.Context, userID string, orgs []UserOrg) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_orgs WHERE pocket_id_user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete user orgs: %w", err)
	}

	for _, o := range orgs {
		isMain := 0
		if o.IsMain {
			isMain = 1
		}
		const q = `INSERT INTO user_orgs (pocket_id_user_id, sid, rank_name, is_main) VALUES (?, ?, ?, ?)`
		if _, err := tx.ExecContext(ctx, q, userID, o.SID, o.RankName, isMain); err != nil {
			return fmt.Errorf("insert user org: %w", err)
		}
	}

	return tx.Commit()
}

// GetUserOrgs returns all org memberships for a user, joined with cached org info.
// Orgs without a cache entry are included with an empty name/logo.
type UserOrgDetail struct {
	SID      string
	Name     string
	LogoPath string
	RankName string
	IsMain   bool
}

func (s *Store) GetUserOrgs(ctx context.Context, userID string) ([]UserOrgDetail, error) {
	const q = `
SELECT uo.sid, COALESCE(oc.name,''), COALESCE(oc.logo_path,''), uo.rank_name, uo.is_main
FROM user_orgs uo
LEFT JOIN org_cache oc ON oc.sid = uo.sid
WHERE uo.pocket_id_user_id = ?
ORDER BY uo.is_main DESC, uo.sid ASC`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("query user orgs: %w", err)
	}
	defer rows.Close()

	var result []UserOrgDetail
	for rows.Next() {
		var d UserOrgDetail
		var isMain int
		if err := rows.Scan(&d.SID, &d.Name, &d.LogoPath, &d.RankName, &isMain); err != nil {
			return nil, fmt.Errorf("scan user org: %w", err)
		}
		d.IsMain = isMain == 1
		result = append(result, d)
	}
	return result, rows.Err()
}
