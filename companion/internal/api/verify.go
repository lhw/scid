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

	if time.Now().After(vt.ExpiresAt) {
		writeError(w, http.StatusBadRequest, "verification token has expired; please start a new verification")
		return
	}

	profile, err := s.scraper.FetchProfile(r.Context(), vt.RSIHandle)
	if err != nil {
		slog.WarnContext(r.Context(), "RSI profile fetch failed",
			"handle", vt.RSIHandle, "err", err)
		writeError(w, http.StatusBadGateway, "could not fetch RSI profile")
		return
	}

	if !rsi.ContainsToken(profile, vt.Token) {
		writeJSON(w, http.StatusOK, confirmResponse{
			Verified: false,
			Message:  "token not found in bio",
		})
		return
	}

	if err := s.completeVerification(r.Context(), user.ID, vt, profile); err != nil {
		slog.ErrorContext(r.Context(), "complete verification failed",
			"user_id", user.ID, "handle", vt.RSIHandle, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

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
		{Key: "rsi_citizen_record", Value: profile.CitizenRecord},
		{Key: "rsi_enlisted", Value: profile.Enlisted},
	}
	if err := s.pid.SetCustomClaims(ctx, userID, claims); err != nil {
		return fmt.Errorf("set custom claims: %w", err)
	}

	return nil
}

// --- GET /api/verify/status ---

type statusResponse struct {
	Verified         bool       `json:"verified"`
	Handle           string     `json:"handle,omitempty"`
	VerifiedAt       string     `json:"verified_at,omitempty"`
	PendingHandle    string     `json:"pending_handle,omitempty"`
	PendingExpiresAt *time.Time `json:"pending_expires_at,omitempty"`
}

func (s *Server) handleVerifyStatus(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	resp := statusResponse{}

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
			break
		}
	}

	// NOTE: rsi_handle is surfaced in OIDC token claims. A future iteration can
	// add GET /api/custom-claims/user/:id to the Pocket ID client to also
	// populate resp.Handle and resp.VerifiedAt here.

	writeJSON(w, http.StatusOK, resp)
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
