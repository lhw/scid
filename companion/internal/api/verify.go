package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/lhw/scid/companion/internal/pocketid"
	"github.com/lhw/scid/companion/internal/rsi"
	"github.com/lhw/scid/companion/internal/store"
)

// tokenExpiry is how long a verification token remains valid.
const tokenExpiry = 24 * time.Hour

// handleRegexp validates RSI handles: alphanumeric, hyphens, underscores, 3-60 chars.
var handleRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,60}$`)

// contextKey is the unexported type for context keys in this package.
type contextKey int

const userContextKey contextKey = iota

// withUser stores a Pocket ID user in a context.
func withUser(ctx context.Context, u *pocketid.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

// userFromContext retrieves the Pocket ID user stored by bearerAuthMiddleware.
func userFromContext(ctx context.Context) *pocketid.User {
	u, _ := ctx.Value(userContextKey).(*pocketid.User)
	return u
}

// --- POST /api/verify/start ---

type startRequest struct {
	Handle string `json:"handle"`
}

type startResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Server) handleVerifyStart(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	var req startRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if !handleRegexp.MatchString(req.Handle) {
		writeError(w, http.StatusBadRequest, "handle must be 3-60 alphanumeric characters, hyphens, or underscores")
		return
	}

	linked, err := s.store.HandleIsLinkedToOtherUser(r.Context(), req.Handle, user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "handle uniqueness check failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if linked {
		writeError(w, http.StatusConflict, "this RSI handle is already linked to another account")
		return
	}

	token, err := generateToken()
	if err != nil {
		slog.ErrorContext(r.Context(), "token generation failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	now := time.Now().UTC()
	vt := &store.VerificationToken{
		ID:             newID(),
		PocketIDUserID: user.ID,
		RSIHandle:      req.Handle,
		Token:          token,
		CreatedAt:      now,
		ExpiresAt:      now.Add(tokenExpiry),
	}

	saved, err := s.store.UpsertToken(r.Context(), vt)
	if err != nil {
		slog.ErrorContext(r.Context(), "upsert token failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, startResponse{
		Token:     saved.Token,
		ExpiresAt: saved.ExpiresAt,
	})
}

// --- POST /api/verify/confirm ---

type confirmResponse struct {
	Verified bool   `json:"verified"`
	Handle   string `json:"handle,omitempty"`
	Message  string `json:"message,omitempty"`
}

func (s *Server) handleVerifyConfirm(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	vt, err := s.store.GetTokenByUserID(r.Context(), user.ID)
	if err == store.ErrNotFound {
		writeError(w, http.StatusBadRequest, "no pending verification found; call /api/verify/start first")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "get token failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if time.Now().UTC().After(vt.ExpiresAt.UTC()) {
		slog.WarnContext(r.Context(), "verification token expired",
			"user_id", user.ID,
			"expires_at", vt.ExpiresAt.UTC(),
			"now", time.Now().UTC())
		writeError(w, http.StatusBadRequest, "verification token has expired; please start a new verification")
		return
	}

	profile, err := s.scraper.FetchProfile(r.Context(), vt.RSIHandle)
	if err != nil {
		slog.WarnContext(r.Context(), "RSI profile fetch failed",
			"handle", vt.RSIHandle, "err", err)
		metricVerifications.WithLabelValues("fetch_error").Inc()
		writeError(w, http.StatusBadGateway, "could not fetch RSI profile")
		return
	}

	if !rsi.ContainsToken(profile, vt.Token) {
		auditLog(r.Context(), "rsi.verify.failed",
			"user_id", user.ID, "handle", vt.RSIHandle, "reason", "token_not_found")
		metricVerifications.WithLabelValues("token_mismatch").Inc()
		writeJSON(w, http.StatusOK, confirmResponse{
			Verified: false,
			Message:  "token not found in bio",
		})
		return
	}

	if err := s.completeVerification(r.Context(), user.ID, vt, profile); err != nil {
		slog.ErrorContext(r.Context(), "complete verification failed",
			"user_id", user.ID, "handle", vt.RSIHandle, "err", err)
		metricVerifications.WithLabelValues("error").Inc()
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	auditLog(r.Context(), "rsi.verified", "user_id", user.ID, "handle", vt.RSIHandle)
	metricVerifications.WithLabelValues("success").Inc()

	if err := s.store.DeleteTokenByUserID(r.Context(), user.ID); err != nil {
		slog.WarnContext(r.Context(), "delete token failed", "user_id", user.ID, "err", err)
	}

	writeJSON(w, http.StatusOK, confirmResponse{
		Verified: true,
		Handle:   vt.RSIHandle,
	})
}

func (s *Server) completeVerification(
	ctx context.Context,
	userID string,
	vt *store.VerificationToken,
	profile *rsi.Profile,
) error {
	verifiedGroup, err := s.pid.EnsureGroupExists(ctx, "verified", "Verified RSI Citizens")
	if err != nil {
		return fmt.Errorf("ensure verified group: %w", err)
	}

	currentGroups, err := s.pid.GetUserGroups(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user groups: %w", err)
	}

	groupIDs := make([]string, 0, len(currentGroups)+1)
	alreadyVerified := false
	for _, g := range currentGroups {
		groupIDs = append(groupIDs, g.ID)
		if g.ID == verifiedGroup.ID {
			alreadyVerified = true
		}
	}
	if !alreadyVerified {
		groupIDs = append(groupIDs, verifiedGroup.ID)
	}

	if err := s.pid.SetUserGroups(ctx, userID, groupIDs); err != nil {
		return fmt.Errorf("set user groups: %w", err)
	}

	claims := []pocketid.CustomClaim{
		{Key: "rsi_handle", Value: vt.RSIHandle},
		{Key: "rsi_verified_at", Value: time.Now().UTC().Format(time.RFC3339)},
		{Key: "rsi_enlisted", Value: parseEnlistDate(profile.Enlisted)},
	}
	if r := strings.TrimSpace(profile.CitizenRecord); r != "" && strings.ToLower(r) != "n/a" {
		claims = append(claims, pocketid.CustomClaim{Key: "rsi_citizen_record", Value: r})
	}
	if err := s.pid.SetCustomClaims(ctx, userID, claims); err != nil {
		return fmt.Errorf("set custom claims: %w", err)
	}

	if profile.AvatarURL != "" {
		if err := s.pid.SetProfilePicture(ctx, userID, profile.AvatarURL); err != nil {
			// Non-fatal: log and continue without failing the whole verification.
			slog.WarnContext(ctx, "failed to upload profile picture", "err", err)
		}
	}

	// Sync org memberships in the background (non-fatal).
	go s.syncUserOrgs(context.Background(), userID, vt.RSIHandle)

	// Seed the background org re-sync schedule so the job knows when this
	// user is next due. synced_at = now means the first background sync fires
	// ~7 days from today.
	if err := s.store.UpsertOrgSync(context.Background(), userID, vt.RSIHandle, time.Now()); err != nil {
		slog.Warn("verify: seed org sync entry", "user_id", userID, "err", err)
	}

	return nil
}

// parseEnlistDate converts an RSI enlistment date string like "Apr 16, 2023"
// into ISO 8601 date format "2023-04-16". Returns the original string if it
// cannot be parsed.
func parseEnlistDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Try the format RSI uses: "Jan 2, 2006"
	if t, err := time.Parse("Jan 2, 2006", s); err == nil {
		return t.Format("2006-01-02")
	}
	// Already looks like ISO format
	if len(s) == 10 && s[4] == '-' {
		return s
	}
	return s
}

// --- GET /api/verify/status ---

type statusResponse struct {
	Authenticated    bool       `json:"authenticated"`
	UserID           string     `json:"user_id,omitempty"`
	Username         string     `json:"username,omitempty"`
	Verified         bool       `json:"verified"`
	Admin            bool       `json:"admin,omitempty"`
	Handle           string     `json:"handle,omitempty"`
	VerifiedAt       string     `json:"verified_at,omitempty"`
	Enlisted         string     `json:"enlisted,omitempty"`
	CitizenRecord    string     `json:"citizen_record,omitempty"`
	Orgs             []OrgEntry `json:"orgs,omitempty"`
	PendingHandle    string     `json:"pending_handle,omitempty"`
	PendingExpiresAt *time.Time `json:"pending_expires_at,omitempty"`
	NextSyncAt       *time.Time `json:"next_sync_at,omitempty"`
}

func (s *Server) handleVerifyStatus(w http.ResponseWriter, r *http.Request) {
	// If no valid Bearer token, return an unauthenticated-but-OK response so
	// the home page can render without the user being logged in.
	token := extractBearerToken(r)
	if token == "" {
		writeJSON(w, http.StatusOK, statusResponse{})
		return
	}
	user, err := s.pid.GetCurrentUser(r.Context(), token)
	if err != nil {
		writeJSON(w, http.StatusOK, statusResponse{})
		return
	}

	resp := statusResponse{
		Authenticated: true,
		UserID:        user.ID,
		Username:      user.Username,
	}

	pending, err := s.store.GetTokenByUserID(r.Context(), user.ID)
	if err != nil && err != store.ErrNotFound {
		slog.ErrorContext(r.Context(), "get pending token failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if pending != nil && pending.ExpiresAt.After(time.Now()) {
		resp.PendingHandle = pending.RSIHandle
		exp := pending.ExpiresAt
		resp.PendingExpiresAt = &exp
	}

	groups, err := s.pid.GetUserGroups(r.Context(), user.ID)
	if err != nil {
		slog.WarnContext(r.Context(), "get user groups failed", "err", err)
		writeJSON(w, http.StatusOK, resp)
		return
	}

	for _, g := range groups {
		if g.Name == "verified" {
			resp.Verified = true
		}
		if g.Name == "admin" {
			resp.Admin = true
		}
	}

	if resp.Verified {
		// Populate claim fields from Pocket ID so the frontend can show the
		// full profile without a second round-trip.
		detail, err := s.pid.GetUser(r.Context(), user.ID)
		if err != nil {
			slog.WarnContext(r.Context(), "get user detail failed", "err", err)
		} else {
			for _, c := range detail.CustomClaims {
				switch c.Key {
				case "rsi_handle":
					resp.Handle = c.Value
				case "rsi_verified_at":
					resp.VerifiedAt = c.Value
				case "rsi_enlisted":
					resp.Enlisted = c.Value
				case "rsi_citizen_record":
					resp.CitizenRecord = c.Value
				}
			}
		}

		// Populate org memberships from the local DB.
		if userOrgs, err := s.store.GetUserOrgs(r.Context(), user.ID); err == nil {
			for _, o := range userOrgs {
				resp.Orgs = append(resp.Orgs, OrgEntry{
					SID:      o.SID,
					Name:     o.Name,
					RankName: o.RankName,
					IsMain:   o.IsMain,
					HasLogo:  o.LogoPath != "",
				})
			}
		} else {
			slog.WarnContext(r.Context(), "get user orgs failed", "err", err)
		}

		// Expose when the next background org re-sync is due.
		if syncEntry, err := s.store.GetOrgSync(r.Context(), user.ID); err == nil {
			next := syncEntry.SyncedAt.Add(orgReverifyAge)
			resp.NextSyncAt = &next
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- POST /api/verify/refresh ---

func (s *Server) handleVerifyRefresh(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	detail, err := s.pid.GetUser(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get user detail for refresh", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var handle, verifiedAt string
	for _, c := range detail.CustomClaims {
		switch c.Key {
		case "rsi_handle":
			handle = c.Value
		case "rsi_verified_at":
			verifiedAt = c.Value
		}
	}
	if handle == "" {
		writeError(w, http.StatusBadRequest, "account not verified")
		return
	}

	profile, err := s.scraper.FetchProfile(r.Context(), handle)
	if err != nil {
		slog.ErrorContext(r.Context(), "refresh: fetch RSI profile", "handle", handle, "err", err)
		writeError(w, http.StatusBadGateway, "could not fetch RSI profile")
		return
	}

	claims := []pocketid.CustomClaim{
		{Key: "rsi_handle", Value: handle},
		{Key: "rsi_verified_at", Value: verifiedAt},
		{Key: "rsi_enlisted", Value: parseEnlistDate(profile.Enlisted)},
	}
	if cr := strings.TrimSpace(profile.CitizenRecord); cr != "" && strings.ToLower(cr) != "n/a" {
		claims = append(claims, pocketid.CustomClaim{Key: "rsi_citizen_record", Value: cr})
	}

	if err := s.pid.SetCustomClaims(r.Context(), user.ID, claims); err != nil {
		slog.ErrorContext(r.Context(), "refresh: set custom claims", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if profile.AvatarURL != "" {
		if err := s.pid.SetProfilePicture(r.Context(), user.ID, profile.AvatarURL); err != nil {
			slog.WarnContext(r.Context(), "refresh: failed to upload profile picture", "err", err)
		}
	}

	// Sync org memberships synchronously during refresh so the response includes up-to-date orgs.
	s.syncUserOrgs(r.Context(), user.ID, handle)

	// Reset the re-sync timer so the background job won't fire again for 7 days.
	if err := s.store.UpsertOrgSync(r.Context(), user.ID, handle, time.Now()); err != nil {
		slog.Warn("verify: update org sync entry after refresh", "user_id", user.ID, "err", err)
	}

	// Return the same shape as GET /api/verify/status for easy consumption.
	resp := statusResponse{
		Authenticated: true,
		UserID:        user.ID,
		Username:      user.Username,
		Verified:      true,
		Handle:        handle,
		VerifiedAt:    verifiedAt,
		Enlisted:      parseEnlistDate(profile.Enlisted),
	}
	if cr := strings.TrimSpace(profile.CitizenRecord); cr != "" && strings.ToLower(cr) != "n/a" {
		resp.CitizenRecord = cr
	}
	if userOrgs, err := s.store.GetUserOrgs(r.Context(), user.ID); err == nil {
		for _, o := range userOrgs {
			resp.Orgs = append(resp.Orgs, OrgEntry{
				SID:      o.SID,
				Name:     o.Name,
				RankName: o.RankName,
				IsMain:   o.IsMain,
				HasLogo:  o.LogoPath != "",
			})
		}
	}
	nextSync := time.Now().Add(orgReverifyAge)
	resp.NextSyncAt = &nextSync
	writeJSON(w, http.StatusOK, resp)
}

// --- POST /api/account/delete ---

func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	// Delete the user from Pocket ID. This is irreversible.
	if err := s.pid.DeleteUser(r.Context(), user.ID); err != nil {
		slog.ErrorContext(r.Context(), "delete account: pocket id deletion failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	slog.InfoContext(r.Context(), "account deleted", "user_id", user.ID)
	auditLog(r.Context(), "account.deleted", "user_id", user.ID, "username", user.Username)
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

// generateToken creates a new verification token: scid:<16 random hex bytes>.
func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return "scid:" + hex.EncodeToString(b), nil
}

// newID generates a random hex ID for a DB row.
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}
