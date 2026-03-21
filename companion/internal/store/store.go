package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
-- status: 'pending' | 'approved' | 'rejected'
CREATE TABLE IF NOT EXISTS app_registrations (
    id TEXT PRIMARY KEY,
    oidc_client_id TEXT NOT NULL UNIQUE,
    owner_user_id TEXT NOT NULL,
    verified_only INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- user_org_sync tracks when each verified user's RSI org memberships were
-- last synced by the background re-verify job. synced_at is initialised to
-- the user's rsi_verified_at timestamp so the initial crop of re-syncs is
-- spread out over time rather than all firing at once.
CREATE TABLE IF NOT EXISTS user_org_sync (
    pocket_id_user_id TEXT PRIMARY KEY,
    handle TEXT NOT NULL,
    synced_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
`

// migrations adds columns that were introduced after initial schema creation.
// Each statement is expected to fail silently if the column already exists.
const migrations = `
ALTER TABLE app_registrations ADD COLUMN status TEXT NOT NULL DEFAULT 'approved';
ALTER TABLE app_registrations ADD COLUMN rejection_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE app_registrations ADD COLUMN listed INTEGER NOT NULL DEFAULT 0;
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
	// Run incremental migrations — ignore errors for already-applied statements
	// (SQLite returns an error when trying to add a column that already exists).
	for _, stmt := range splitMigrations(migrations) {
		db.Exec(stmt) // nolint:errcheck — intentionally ignoring re-run errors
	}
	return &Store{db: db}, nil
}

// splitMigrations splits a multi-statement migration string on semicolons.
func splitMigrations(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
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
	ID              string
	OIDCClientID    string
	OwnerUserID     string
	VerifiedOnly    bool
	Listed          bool   // listed in the public app directory
	Status          string // "pending" | "approved" | "rejected"
	RejectionReason string
	CreatedAt       time.Time
}

// CreateAppRegistration inserts a new app registration row.
func (s *Store) CreateAppRegistration(ctx context.Context, reg *AppRegistration) error {
	const q = `
INSERT INTO app_registrations (id, oidc_client_id, owner_user_id, verified_only, listed, status, rejection_reason, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	verifiedOnly := 0
	if reg.VerifiedOnly {
		verifiedOnly = 1
	}
	listed := 0
	if reg.Listed {
		listed = 1
	}
	status := reg.Status
	if status == "" {
		status = "approved"
	}
	_, err := s.db.ExecContext(ctx, q,
		reg.ID, reg.OIDCClientID, reg.OwnerUserID, verifiedOnly, listed, status, reg.RejectionReason,
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
SELECT id, oidc_client_id, owner_user_id, verified_only, COALESCE(listed,0), COALESCE(status,'approved'), COALESCE(rejection_reason,''), created_at
FROM app_registrations
WHERE oidc_client_id = ?`
	return scanAppRegistration(s.db.QueryRowContext(ctx, q, oidcClientID))
}

// ListAppRegistrationsByOwner returns all app registrations for a given user.
func (s *Store) ListAppRegistrationsByOwner(ctx context.Context, ownerUserID string) ([]AppRegistration, error) {
	const q = `
SELECT id, oidc_client_id, owner_user_id, verified_only, COALESCE(listed,0), COALESCE(status,'approved'), COALESCE(rejection_reason,''), created_at
FROM app_registrations
WHERE owner_user_id = ?
ORDER BY created_at ASC`
	return queryAppRegistrations(s.db, ctx, q, ownerUserID)
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

// UpdateAppRegistrationStatus updates the status (and optional rejection reason) for an app.
func (s *Store) UpdateAppRegistrationStatus(ctx context.Context, oidcClientID, status, rejectionReason string) error {
	const q = `UPDATE app_registrations SET status = ?, rejection_reason = ? WHERE oidc_client_id = ?`
	if _, err := s.db.ExecContext(ctx, q, status, rejectionReason, oidcClientID); err != nil {
		return fmt.Errorf("update app registration status: %w", err)
	}
	return nil
}

// UpdateAppRegistrationListed updates the listed flag for an app.
func (s *Store) UpdateAppRegistrationListed(ctx context.Context, oidcClientID string, listed bool) error {
	flag := 0
	if listed {
		flag = 1
	}
	const q = `UPDATE app_registrations SET listed = ? WHERE oidc_client_id = ?`
	if _, err := s.db.ExecContext(ctx, q, flag, oidcClientID); err != nil {
		return fmt.Errorf("update app registration listed: %w", err)
	}
	return nil
}

// ListListedApps returns all approved apps that have opted into the public directory.
func (s *Store) ListListedApps(ctx context.Context) ([]AppRegistration, error) {
	const q = `
SELECT id, oidc_client_id, owner_user_id, verified_only, COALESCE(listed,0), COALESCE(status,'approved'), COALESCE(rejection_reason,''), created_at
FROM app_registrations
WHERE COALESCE(listed,0) = 1 AND COALESCE(status,'approved') = 'approved'
ORDER BY created_at ASC`
	return queryAppRegistrations(s.db, ctx, q)
}

// ListPendingAppRegistrations returns all app registrations with status='pending'.
func (s *Store) ListPendingAppRegistrations(ctx context.Context) ([]AppRegistration, error) {
	const q = `
SELECT id, oidc_client_id, owner_user_id, verified_only, COALESCE(listed,0), COALESCE(status,'approved'), COALESCE(rejection_reason,''), created_at
FROM app_registrations
WHERE COALESCE(status,'approved') = 'pending'
ORDER BY created_at ASC`
	return queryAppRegistrations(s.db, ctx, q)
}

// ListAllAppRegistrations returns all app registrations ordered by creation date.
func (s *Store) ListAllAppRegistrations(ctx context.Context) ([]AppRegistration, error) {
	const q = `
SELECT id, oidc_client_id, owner_user_id, verified_only, COALESCE(listed,0), COALESCE(status,'approved'), COALESCE(rejection_reason,''), created_at
FROM app_registrations
ORDER BY created_at DESC`
	return queryAppRegistrations(s.db, ctx, q)
}

func queryAppRegistrations(db *sql.DB, ctx context.Context, query string, args ...any) ([]AppRegistration, error) {
	rows, err := db.QueryContext(ctx, query, args...)
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

func scanAppRegistration(row *sql.Row) (*AppRegistration, error) {
	var reg AppRegistration
	var verifiedOnly, listed int
	var createdAt string
	err := row.Scan(&reg.ID, &reg.OIDCClientID, &reg.OwnerUserID, &verifiedOnly, &listed, &reg.Status, &reg.RejectionReason, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan app registration: %w", err)
	}
	reg.VerifiedOnly = verifiedOnly == 1
	reg.Listed = listed == 1
	reg.CreatedAt, _ = parseTime(createdAt)
	return &reg, nil
}

func scanAppRegistrationRow(rows *sql.Rows) (*AppRegistration, error) {
	var reg AppRegistration
	var verifiedOnly, listed int
	var createdAt string
	err := rows.Scan(&reg.ID, &reg.OIDCClientID, &reg.OwnerUserID, &verifiedOnly, &listed, &reg.Status, &reg.RejectionReason, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan app registration: %w", err)
	}
	reg.VerifiedOnly = verifiedOnly == 1
	reg.Listed = listed == 1
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

// OrgSyncEntry records the last time a user's RSI org memberships were synced.
type OrgSyncEntry struct {
	PocketIDUserID string
	Handle         string
	SyncedAt       time.Time
}

// UpsertOrgSync sets (or updates) the org sync timestamp for a user.
func (s *Store) UpsertOrgSync(ctx context.Context, userID, handle string, syncedAt time.Time) error {
	const q = `INSERT OR REPLACE INTO user_org_sync (pocket_id_user_id, handle, synced_at) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, userID, handle, syncedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert org sync: %w", err)
	}
	return nil
}

// InsertOrgSyncIfMissing seeds a sync entry only if none exists for this user.
// Used during startup to enroll existing verified users; INSERT OR IGNORE leaves
// already-scheduled users undisturbed.
func (s *Store) InsertOrgSyncIfMissing(ctx context.Context, userID, handle string, syncedAt time.Time) error {
	const q = `INSERT OR IGNORE INTO user_org_sync (pocket_id_user_id, handle, synced_at) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, userID, handle, syncedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("insert org sync: %w", err)
	}
	return nil
}

// GetOrgSync returns the sync entry for a specific user, or ErrNotFound.
func (s *Store) GetOrgSync(ctx context.Context, userID string) (OrgSyncEntry, error) {
	const q = `SELECT pocket_id_user_id, handle, synced_at FROM user_org_sync WHERE pocket_id_user_id = ?`
	row := s.db.QueryRowContext(ctx, q, userID)
	var e OrgSyncEntry
	var syncedAt string
	if err := row.Scan(&e.PocketIDUserID, &e.Handle, &syncedAt); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return OrgSyncEntry{}, ErrNotFound
		}
		return OrgSyncEntry{}, fmt.Errorf("get org sync: %w", err)
	}
	e.SyncedAt, _ = parseTime(syncedAt)
	return e, nil
}

// ListExpiredOrgSyncs returns all entries where synced_at is older than cutoff,
// ordered oldest-first so the most-stale users are processed first.
func (s *Store) ListExpiredOrgSyncs(ctx context.Context, cutoff time.Time) ([]OrgSyncEntry, error) {
	const q = `
SELECT pocket_id_user_id, handle, synced_at
FROM user_org_sync
WHERE synced_at < ?
ORDER BY synced_at ASC`
	rows, err := s.db.QueryContext(ctx, q, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("query expired org syncs: %w", err)
	}
	defer rows.Close()

	var result []OrgSyncEntry
	for rows.Next() {
		var e OrgSyncEntry
		var syncedAt string
		if err := rows.Scan(&e.PocketIDUserID, &e.Handle, &syncedAt); err != nil {
			return nil, fmt.Errorf("scan org sync: %w", err)
		}
		e.SyncedAt, _ = parseTime(syncedAt)
		result = append(result, e)
	}
	return result, rows.Err()
}

// Ping verifies the SQLite connection is alive by running a trivial query.
func (s *Store) Ping(ctx context.Context) error {
	row := s.db.QueryRowContext(ctx, "SELECT 1")
	var n int
	return row.Scan(&n)
}
