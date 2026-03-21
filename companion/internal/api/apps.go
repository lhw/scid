package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/scid/companion/internal/pocketid"
	"github.com/lhw/scid/companion/internal/store"
)

// maxLogoSize is the maximum accepted logo upload size (1 MiB).
const maxLogoSize = 1 << 20

// maxAppsPerUser is how many OIDC clients a single verified user may register.
const maxAppsPerUser = 5

// --- shared request/response types ---

type createAppRequest struct {
	Name         string   `json:"name"`
	LaunchURL    string   `json:"launch_url"`
	RedirectURIs []string `json:"redirect_uris"`
	LogoutURIs   []string `json:"logout_uris"`
	IsPublic     bool     `json:"is_public"`
	PkceRequired bool     `json:"pkce_required"`
	VerifiedOnly bool     `json:"verified_only"`
}

type appResponse struct {
	ID           string   `json:"id"`
	ClientSecret string   `json:"client_secret,omitempty"`
	Name         string   `json:"name"`
	LaunchURL    string   `json:"launch_url,omitempty"`
	RedirectURIs []string `json:"redirect_uris"`
	LogoutURIs   []string `json:"logout_uris"`
	IsPublic     bool     `json:"is_public"`
	PkceRequired bool     `json:"pkce_required"`
	VerifiedOnly bool     `json:"verified_only"`
	HasLogo      bool     `json:"has_logo"`
	CreatedAt    string   `json:"created_at"`
}

// buildAppResponse merges an OIDCClient from Pocket ID with a store.AppRegistration.
func buildAppResponse(client *pocketid.OIDCClient, reg *store.AppRegistration, secret string) appResponse {
	launchURL := ""
	if client.LaunchURL != nil {
		launchURL = *client.LaunchURL
	}
	redirectURIs := client.CallbackURLs
	if redirectURIs == nil {
		redirectURIs = []string{}
	}
	logoutURIs := client.LogoutCallbackURLs
	if logoutURIs == nil {
		logoutURIs = []string{}
	}
	return appResponse{
		ID:           client.ID,
		ClientSecret: secret,
		Name:         client.Name,
		LaunchURL:    launchURL,
		RedirectURIs: redirectURIs,
		LogoutURIs:   logoutURIs,
		IsPublic:     client.IsPublic,
		PkceRequired: client.PkceEnabled,
		VerifiedOnly: reg.VerifiedOnly,
		HasLogo:      client.HasLogo,
		CreatedAt:    reg.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// --- validation ---

func validateCreateAppRequest(req createAppRequest) string {
	name := strings.TrimSpace(req.Name)
	if len(name) == 0 {
		return "name is required"
	}
	if len(name) > 50 {
		return "name must be 50 characters or fewer"
	}
	if req.LaunchURL != "" {
		if !strings.HasPrefix(req.LaunchURL, "https://") {
			return "launch_url must be an https:// URL"
		}
	}
	if len(req.RedirectURIs) == 0 {
		return "at least one redirect_uri is required"
	}
	if len(req.RedirectURIs) > 10 {
		return "at most 10 redirect_uris are allowed"
	}
	for _, u := range req.RedirectURIs {
		if !isValidCallbackURL(u) {
			return "redirect_uri must be an https:// URL or http://localhost/127.0.0.1 URL; wildcards are not allowed"
		}
	}
	if len(req.LogoutURIs) > 10 {
		return "at most 10 logout_uris are allowed"
	}
	for _, u := range req.LogoutURIs {
		if !isValidCallbackURL(u) {
			return "logout_uri must be an https:// URL or http://localhost/127.0.0.1 URL; wildcards are not allowed"
		}
	}
	return ""
}

// isValidCallbackURL returns true for https:// URLs and http://localhost or 127.0.0.1 URLs.
// Wildcards are explicitly rejected.
func isValidCallbackURL(u string) bool {
	if strings.Contains(u, "*") {
		return false
	}
	if strings.HasPrefix(u, "https://") {
		return true
	}
	if strings.HasPrefix(u, "http://localhost") || strings.HasPrefix(u, "http://127.0.0.1") {
		return true
	}
	return false
}

// buildOIDCClientParams maps a createAppRequest to Pocket ID params.
func buildOIDCClientParams(req createAppRequest) pocketid.OIDCClientParams {
	var launchURL *string
	if req.LaunchURL != "" {
		u := req.LaunchURL
		launchURL = &u
	}
	logoutURIs := req.LogoutURIs
	if logoutURIs == nil {
		logoutURIs = []string{}
	}
	return pocketid.OIDCClientParams{
		Name:               strings.TrimSpace(req.Name),
		LaunchURL:          launchURL,
		IsPublic:           req.IsPublic,
		PkceEnabled:        req.PkceRequired,
		CallbackURLs:       req.RedirectURIs,
		LogoutCallbackURLs: logoutURIs,
	}
}

// isUserVerified checks whether the user is in the "verified" Pocket ID group.
func (s *Server) isUserVerified(r *http.Request, userID string) (bool, error) {
	groups, err := s.pid.GetUserGroups(r.Context(), userID)
	if err != nil {
		return false, err
	}
	for _, g := range groups {
		if g.Name == "verified" {
			return true, nil
		}
	}
	return false, nil
}

// ownerRegOrNil looks up an AppRegistration and returns it only if the
// specified user is the owner. Returns nil without error if not found or not owned,
// so callers can send 404 without revealing existence.
func (s *Server) ownerRegOrNil(r *http.Request, clientID, userID string) (*store.AppRegistration, error) {
	reg, err := s.store.GetAppRegistrationByClientID(r.Context(), clientID)
	if err == store.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if reg.OwnerUserID != userID {
		return nil, nil
	}
	return reg, nil
}

// setVerifiedOnlyGroups configures Pocket ID group restriction for verified_only.
// verifiedOnly=true → allows only the "verified" group; false → removes restriction.
func (s *Server) setVerifiedOnlyGroups(r *http.Request, clientID string, verifiedOnly bool) error {
	var groupIDs []string
	if verifiedOnly {
		g, err := s.pid.EnsureGroupExists(r.Context(), "verified", "Verified RSI Citizens")
		if err != nil {
			return err
		}
		groupIDs = []string{g.ID}
	}
	_, err := s.pid.SetOIDCClientAllowedGroups(r.Context(), clientID, groupIDs)
	return err
}

// --- POST /api/apps ---

func (s *Server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	verified, err := s.isUserVerified(r, user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "check verified failed", "user_id", user.ID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !verified {
		writeError(w, http.StatusForbidden, "only verified RSI citizens can register applications")
		return
	}

	regs, err := s.store.ListAppRegistrationsByOwner(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list apps failed", "user_id", user.ID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if len(regs) >= maxAppsPerUser {
		writeError(w, http.StatusUnprocessableEntity, "maximum of 5 applications per user reached")
		return
	}

	var req createAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if msg := validateCreateAppRequest(req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	params := buildOIDCClientParams(req)
	client, err := s.pid.CreateOIDCClient(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "create oidc client failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// For private clients, generate the initial secret immediately.
	var secret string
	if !req.IsPublic {
		secret, err = s.pid.RotateOIDCClientSecret(r.Context(), client.ID)
		if err != nil {
			slog.ErrorContext(r.Context(), "initial secret rotation failed", "client_id", client.ID, "err", err)
			// Best-effort cleanup; ignore error.
			_ = s.pid.DeleteOIDCClient(r.Context(), client.ID)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	if req.VerifiedOnly {
		if err := s.setVerifiedOnlyGroups(r, client.ID, true); err != nil {
			slog.ErrorContext(r.Context(), "set verified-only groups failed", "client_id", client.ID, "err", err)
			_ = s.pid.DeleteOIDCClient(r.Context(), client.ID)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	reg := &store.AppRegistration{
		ID:           newID(),
		OIDCClientID: client.ID,
		OwnerUserID:  user.ID,
		VerifiedOnly: req.VerifiedOnly,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.store.CreateAppRegistration(r.Context(), reg); err != nil {
		slog.ErrorContext(r.Context(), "store app registration failed", "client_id", client.ID, "err", err)
		_ = s.pid.DeleteOIDCClient(r.Context(), client.ID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, buildAppResponse(client, reg, secret))
}

// --- GET /api/apps ---

func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	regs, err := s.store.ListAppRegistrationsByOwner(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list apps failed", "user_id", user.ID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	apps := make([]appResponse, 0, len(regs))
	for _, reg := range regs {
		reg := reg
		client, err := s.pid.GetOIDCClient(r.Context(), reg.OIDCClientID)
		if err != nil {
			slog.WarnContext(r.Context(), "get oidc client failed during list", "client_id", reg.OIDCClientID, "err", err)
			continue
		}
		apps = append(apps, buildAppResponse(client, &reg, ""))
	}

	writeJSON(w, http.StatusOK, apps)
}

// --- GET /api/apps/{id} ---

func (s *Server) handleGetApp(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	clientID := chi.URLParam(r, "id")

	reg, err := s.ownerRegOrNil(r, clientID, user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get app registration failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if reg == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	client, err := s.pid.GetOIDCClient(r.Context(), clientID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get oidc client failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, buildAppResponse(client, reg, ""))
}

// --- PUT /api/apps/{id} ---

func (s *Server) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	clientID := chi.URLParam(r, "id")

	reg, err := s.ownerRegOrNil(r, clientID, user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get app registration failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if reg == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var req createAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if msg := validateCreateAppRequest(req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	params := buildOIDCClientParams(req)
	client, err := s.pid.UpdateOIDCClient(r.Context(), clientID, params)
	if err != nil {
		slog.ErrorContext(r.Context(), "update oidc client failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Update verified_only if it changed.
	if req.VerifiedOnly != reg.VerifiedOnly {
		if err := s.setVerifiedOnlyGroups(r, clientID, req.VerifiedOnly); err != nil {
			slog.ErrorContext(r.Context(), "update verified-only groups failed", "client_id", clientID, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if err := s.store.UpdateAppRegistrationVerifiedOnly(r.Context(), clientID, req.VerifiedOnly); err != nil {
			slog.ErrorContext(r.Context(), "update app registration verified_only failed", "client_id", clientID, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		reg.VerifiedOnly = req.VerifiedOnly
	}

	writeJSON(w, http.StatusOK, buildAppResponse(client, reg, ""))
}

// --- DELETE /api/apps/{id} ---

func (s *Server) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	clientID := chi.URLParam(r, "id")

	reg, err := s.ownerRegOrNil(r, clientID, user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get app registration failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if reg == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if err := s.pid.DeleteOIDCClient(r.Context(), clientID); err != nil {
		slog.ErrorContext(r.Context(), "delete oidc client failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := s.store.DeleteAppRegistrationByClientID(r.Context(), clientID); err != nil {
		slog.ErrorContext(r.Context(), "delete app registration failed", "client_id", clientID, "err", err)
		// Client is already deleted from Pocket ID; log and continue.
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- POST /api/apps/{id}/secret ---

func (s *Server) handleRotateSecret(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	clientID := chi.URLParam(r, "id")

	reg, err := s.ownerRegOrNil(r, clientID, user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get app registration failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if reg == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	secret, err := s.pid.RotateOIDCClientSecret(r.Context(), clientID)
	if err != nil {
		slog.ErrorContext(r.Context(), "rotate secret failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"client_secret": secret})
}

// --- PUT /api/apps/{id}/logo ---

func (s *Server) handleUploadLogo(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	clientID := chi.URLParam(r, "id")

	reg, err := s.ownerRegOrNil(r, clientID, user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get app registration failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if reg == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	// Limit to slightly over maxLogoSize to catch files that are too big.
	if err := r.ParseMultipartForm(maxLogoSize + 1024); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	switch contentType {
	case "image/png", "image/jpeg", "image/webp", "image/svg+xml":
		// accepted
	default:
		writeError(w, http.StatusBadRequest, "unsupported image type; use PNG, JPEG, WebP, or SVG")
		return
	}

	imageData, err := io.ReadAll(io.LimitReader(file, maxLogoSize+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read file")
		return
	}
	if len(imageData) > maxLogoSize {
		writeError(w, http.StatusRequestEntityTooLarge, "logo must be 1 MB or smaller")
		return
	}

	if err := s.pid.SetOIDCClientLogo(r.Context(), clientID, imageData, contentType); err != nil {
		slog.ErrorContext(r.Context(), "upload logo failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
