package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/scid/companion/internal/mailer"
	"github.com/lhw/scid/companion/internal/pocketid"
	"github.com/lhw/scid/companion/internal/store"
)

// validAppName allows alphanumeric characters, spaces, and hyphens only.
var validAppName = regexp.MustCompile(`^[a-zA-Z0-9 -]+$`)

// maxLogoSize is the maximum accepted logo upload size (1 MiB).
const maxLogoSize = 1 << 20

// maxAppsPerUser is how many OIDC clients a single verified user may register.
const maxAppsPerUser = 5

// --- shared request/response types ---

// knownCategories is the exhaustive set of accepted category slugs.
var knownCategories = map[string]bool{
	"community": true,
	"fleet":     true,
	"trading":   true,
	"roleplay":  true,
	"stats":     true,
	"tools":     true,
}

type createAppRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	LaunchURL    string   `json:"launch_url"`
	RedirectURIs []string `json:"redirect_uris"`
	LogoutURIs   []string `json:"logout_uris"`
	IsPublic     bool     `json:"is_public"`
	PkceRequired bool     `json:"pkce_required"`
	VerifiedOnly bool     `json:"verified_only"`
	Listed       bool     `json:"listed"`
	Category     string   `json:"category"`
}

type appResponse struct {
	ID              string   `json:"id"`
	ClientSecret    string   `json:"client_secret,omitempty"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	OwnerUsername   string   `json:"owner_username,omitempty"`
	LaunchURL       string   `json:"launch_url,omitempty"`
	RedirectURIs    []string `json:"redirect_uris"`
	LogoutURIs      []string `json:"logout_uris"`
	IsPublic        bool     `json:"is_public"`
	PkceRequired    bool     `json:"pkce_required"`
	VerifiedOnly    bool     `json:"verified_only"`
	Listed          bool     `json:"listed"`
	HasLogo         bool     `json:"has_logo"`
	Status          string   `json:"status"`
	RejectionReason string   `json:"rejection_reason,omitempty"`
	Category        string   `json:"category,omitempty"`
	CreatedAt       string   `json:"created_at"`
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
	status := reg.Status
	if status == "" {
		status = "approved"
	}
	return appResponse{
		ID:              client.ID,
		ClientSecret:    secret,
		Name:            client.Name,
		Description:     reg.Description,
		LaunchURL:       launchURL,
		RedirectURIs:    redirectURIs,
		LogoutURIs:      logoutURIs,
		IsPublic:        client.IsPublic,
		PkceRequired:    client.PkceEnabled,
		VerifiedOnly:    reg.VerifiedOnly,
		Listed:          reg.Listed,
		HasLogo:         client.HasLogo,
		Status:          status,
		RejectionReason: reg.RejectionReason,
		Category:        reg.Category,
		CreatedAt:       reg.CreatedAt.UTC().Format(time.RFC3339),
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
	if !validAppName.MatchString(name) {
		return "name may only contain letters, digits, spaces, and hyphens"
	}
	if len([]rune(req.Description)) > 200 {
		return "description must be 200 characters or fewer"
	}
	if req.Category != "" && !knownCategories[req.Category] {
		return "invalid category"
	}
	if req.LaunchURL != "" {
		if !strings.HasPrefix(req.LaunchURL, "https://") {
			return "launch_url must be an https:// URL"
		}
	}
	if req.Listed && strings.TrimSpace(req.LaunchURL) == "" {
		return "a launch_url is required to list this app in the directory"
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
func isValidCallbackURL(raw string) bool {
	if strings.Contains(raw, "*") {
		return false
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Opaque != "" || parsed.User != nil || parsed.Fragment != "" {
		return false
	}
	if parsed.Scheme == "" || parsed.Host == "" || parsed.Hostname() == "" {
		return false
	}

	switch parsed.Scheme {
	case "https":
		return true
	case "http":
		hostname := strings.ToLower(parsed.Hostname())
		return hostname == "localhost" || hostname == "127.0.0.1"
	default:
		return false
	}
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

func oidcClientParamsFromClient(client *pocketid.OIDCClient) pocketid.OIDCClientParams {
	var launchURL *string
	if client.LaunchURL != nil {
		u := *client.LaunchURL
		launchURL = &u
	}
	logoutURIs := client.LogoutCallbackURLs
	if logoutURIs == nil {
		logoutURIs = []string{}
	}
	callbackURLs := client.CallbackURLs
	if callbackURLs == nil {
		callbackURLs = []string{}
	}
	return pocketid.OIDCClientParams{
		Name:               client.Name,
		LaunchURL:          launchURL,
		IsPublic:           client.IsPublic,
		PkceEnabled:        client.PkceEnabled,
		CallbackURLs:       callbackURLs,
		LogoutCallbackURLs: logoutURIs,
	}
}

// isUserInGroup checks whether the user is a member of the named Pocket ID group.
func (s *Server) isUserInGroup(r *http.Request, userID, groupName string) (bool, error) {
	groups, err := s.pid.GetUserGroups(r.Context(), userID)
	if err != nil {
		return false, err
	}
	for _, g := range groups {
		if g.Name == groupName {
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

func (s *Server) syncOIDCClientAccess(r *http.Request, clientID, status string, verifiedOnly bool) error {
	policy, err := resolveOIDCClientAccessPolicy(status, verifiedOnly)
	if err != nil {
		return err
	}

	switch policy {
	case oidcClientAccessVerifiedOnly:
		return s.setVerifiedOnlyGroups(r, clientID, true)
	case oidcClientAccessOpen:
		return s.setVerifiedOnlyGroups(r, clientID, false)
	default:
		return fmt.Errorf("unknown oidc access policy %q", policy)
	}
}

const (
	oidcClientAccessVerifiedOnly = "verified-only"
	oidcClientAccessOpen         = "open"
)

func resolveOIDCClientAccessPolicy(status string, verifiedOnly bool) (string, error) {
	switch status {
	case "", "approved", "pending", "rejected":
		if verifiedOnly {
			return oidcClientAccessVerifiedOnly, nil
		}
		return oidcClientAccessOpen, nil
	default:
		return "", fmt.Errorf("unsupported app status %q", status)
	}
}

func detectLogoContentType(imageData []byte) (string, bool) {
	contentType := http.DetectContentType(imageData)
	switch contentType {
	case "image/png", "image/jpeg", "image/webp":
		return contentType, true
	case "application/octet-stream":
		if len(imageData) >= 12 && bytes.Equal(imageData[:4], []byte("RIFF")) && bytes.Equal(imageData[8:12], []byte("WEBP")) {
			return "image/webp", true
		}
	}

	return "", false
}

// --- POST /api/apps ---

func (s *Server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	user := userFromContext(r.Context())

	verified, err := s.isUserInGroup(r, user.ID, "verified")
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

	// Determine initial status and apply group restrictions accordingly.
	status := "approved"
	if req.Listed {
		status = "pending"
	}
	if err := s.syncOIDCClientAccess(r, client.ID, status, req.VerifiedOnly); err != nil {
		slog.ErrorContext(r.Context(), "set initial client access failed", "client_id", client.ID, "status", status, "verified_only", req.VerifiedOnly, "err", err)
		_ = s.pid.DeleteOIDCClient(r.Context(), client.ID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	regID, err := newID()
	if err != nil {
		slog.ErrorContext(r.Context(), "id generation failed", "err", err)
		_ = s.pid.DeleteOIDCClient(r.Context(), client.ID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	reg := &store.AppRegistration{
		ID:           regID,
		OIDCClientID: client.ID,
		OwnerUserID:  user.ID,
		VerifiedOnly: req.VerifiedOnly,
		// Persist listing intent even while pending; the directory already filters
		// for approved apps only.
		Listed:      req.Listed,
		Description: strings.TrimSpace(req.Description),
		Category:    req.Category,
		Status:      status,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.store.CreateAppRegistration(r.Context(), reg); err != nil {
		slog.ErrorContext(r.Context(), "store app registration failed", "client_id", client.ID, "err", err)
		_ = s.pid.DeleteOIDCClient(r.Context(), client.ID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if status == "pending" {
		s.mailer.SendNewAppNotification(mailer.NewAppNotification{
			AppName:      client.Name,
			AppID:        client.ID,
			OwnerHandle:  user.Username,
			RedirectURIs: req.RedirectURIs,
			VerifiedOnly: req.VerifiedOnly,
			Listed:       req.Listed,
			CreatedAt:    reg.CreatedAt,
		})
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
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
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
	if req.Listed && reg.Status != "approved" {
		writeError(w, http.StatusUnprocessableEntity, "app must be approved before listing in the directory")
		return
	}

	previousClient, err := s.pid.GetOIDCClient(r.Context(), clientID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get oidc client before update failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	rollbackOIDCClient := func() {
		if _, revertErr := s.pid.UpdateOIDCClient(r.Context(), clientID, oidcClientParamsFromClient(previousClient)); revertErr != nil {
			slog.ErrorContext(r.Context(), "revert oidc client failed", "client_id", clientID, "err", revertErr)
		}
	}

	params := buildOIDCClientParams(req)
	client, err := s.pid.UpdateOIDCClient(r.Context(), clientID, params)
	if err != nil {
		slog.ErrorContext(r.Context(), "update oidc client failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	verifiedOnlyUpdated := false
	listedUpdated := false
	statusUpdated := false
	previousStatus := reg.Status
	previousRejectionReason := reg.RejectionReason
	rollbackState := func() {
		if listedUpdated {
			if revertErr := s.store.UpdateAppRegistrationListed(r.Context(), clientID, !req.Listed); revertErr != nil {
				slog.ErrorContext(r.Context(), "revert listed failed", "client_id", clientID, "err", revertErr)
			}
		}
		if verifiedOnlyUpdated {
			if revertErr := s.store.UpdateAppRegistrationVerifiedOnly(r.Context(), clientID, !req.VerifiedOnly); revertErr != nil {
				slog.ErrorContext(r.Context(), "revert verified_only failed", "client_id", clientID, "err", revertErr)
			}
			if revertErr := s.syncOIDCClientAccess(r, clientID, previousStatus, !req.VerifiedOnly); revertErr != nil {
				slog.ErrorContext(r.Context(), "revert client access failed", "client_id", clientID, "err", revertErr)
			}
		}
		if statusUpdated {
			if revertErr := s.store.UpdateAppRegistrationStatus(r.Context(), clientID, previousStatus, previousRejectionReason); revertErr != nil {
				slog.ErrorContext(r.Context(), "revert app status failed", "client_id", clientID, "err", revertErr)
			}
			// Re-sync groups unless verifiedOnly rollback already handled it.
			if !verifiedOnlyUpdated {
				if revertErr := s.syncOIDCClientAccess(r, clientID, previousStatus, reg.VerifiedOnly); revertErr != nil {
					slog.ErrorContext(r.Context(), "revert client access after status rollback failed", "client_id", clientID, "err", revertErr)
				}
			}
		}
		rollbackOIDCClient()
	}

	if req.VerifiedOnly != reg.VerifiedOnly {
		if err := s.store.UpdateAppRegistrationVerifiedOnly(r.Context(), clientID, req.VerifiedOnly); err != nil {
			rollbackOIDCClient()
			slog.ErrorContext(r.Context(), "update app registration verified_only failed", "client_id", clientID, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		verifiedOnlyUpdated = true
		if err := s.syncOIDCClientAccess(r, clientID, reg.Status, req.VerifiedOnly); err != nil {
			rollbackState()
			slog.ErrorContext(r.Context(), "sync client access failed after verified_only update", "client_id", clientID, "status", reg.Status, "verified_only", req.VerifiedOnly, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		reg.VerifiedOnly = req.VerifiedOnly
	}

	if req.Listed != reg.Listed {
		if err := s.store.UpdateAppRegistrationListed(r.Context(), clientID, req.Listed); err != nil {
			rollbackState()
			slog.ErrorContext(r.Context(), "update app registration listed failed", "client_id", clientID, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		listedUpdated = true
		reg.Listed = req.Listed

		if req.Listed && reg.Status != "pending" {
			if err := s.store.UpdateAppRegistrationStatus(r.Context(), clientID, "pending", ""); err != nil {
				rollbackState()
				slog.ErrorContext(r.Context(), "update app registration status failed", "client_id", clientID, "err", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}
			statusUpdated = true
			reg.Status = "pending"
			reg.RejectionReason = ""
			// Sync Pocket ID groups for the new pending status (clears any stale scid:pending restriction).
			if !verifiedOnlyUpdated {
				if err := s.syncOIDCClientAccess(r, clientID, "pending", reg.VerifiedOnly); err != nil {
					rollbackState()
					slog.ErrorContext(r.Context(), "sync client access after listing status change failed", "client_id", clientID, "err", err)
					writeError(w, http.StatusInternalServerError, "internal error")
					return
				}
			}
		} else if !req.Listed && reg.Status == "pending" {
			if err := s.store.UpdateAppRegistrationStatus(r.Context(), clientID, "approved", ""); err != nil {
				rollbackState()
				slog.ErrorContext(r.Context(), "update app registration status failed", "client_id", clientID, "err", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}
			statusUpdated = true
			reg.Status = "approved"
			reg.RejectionReason = ""
			// Sync Pocket ID groups for the new approved status (clears any stale scid:pending restriction).
			if !verifiedOnlyUpdated {
				if err := s.syncOIDCClientAccess(r, clientID, "approved", reg.VerifiedOnly); err != nil {
					rollbackState()
					slog.ErrorContext(r.Context(), "sync client access after listing status change failed", "client_id", clientID, "err", err)
					writeError(w, http.StatusInternalServerError, "internal error")
					return
				}
			}
		}
	}

	newDescription := strings.TrimSpace(req.Description)
	if newDescription != reg.Description {
		if err := s.store.UpdateAppRegistrationDescription(r.Context(), clientID, newDescription); err != nil {
			slog.ErrorContext(r.Context(), "update app registration description failed", "client_id", clientID, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		reg.Description = newDescription
	}

	if req.Category != reg.Category {
		if err := s.store.UpdateAppRegistrationCategory(r.Context(), clientID, req.Category); err != nil {
			slog.ErrorContext(r.Context(), "update app registration category failed", "client_id", clientID, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		reg.Category = req.Category
	}

	if statusUpdated && reg.Status == "pending" {
		s.mailer.SendNewAppNotification(mailer.NewAppNotification{
			AppName:      client.Name,
			AppID:        client.ID,
			OwnerHandle:  user.Username,
			RedirectURIs: req.RedirectURIs,
			VerifiedOnly: req.VerifiedOnly,
			Listed:       req.Listed,
			CreatedAt:    reg.CreatedAt,
		})
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

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(io.LimitReader(file, maxLogoSize+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read file")
		return
	}
	if len(imageData) > maxLogoSize {
		writeError(w, http.StatusRequestEntityTooLarge, "logo must be 1 MB or smaller")
		return
	}

	contentType, ok := detectLogoContentType(imageData)
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported image type; use PNG, JPEG, or WebP")
		return
	}

	if err := s.pid.SetOIDCClientLogo(r.Context(), clientID, imageData, contentType); err != nil {
		slog.ErrorContext(r.Context(), "upload logo failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- GET /api/oidc/clients/{id}/logo (public proxy) ---

// handleOIDCClientLogo proxies the OIDC client logo from Pocket ID's internal
// service so the browser fetches it from the same origin as the frontend.
// This avoids cross-origin issues and insulates the browser from Pocket ID's
// internal URL structure.
func (s *Server) handleOIDCClientLogo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	found, err := s.pid.GetOIDCClientLogo(r.Context(), id, w)
	if err != nil {
		slog.ErrorContext(r.Context(), "proxy logo failed", "client_id", id, "err", err)
		// Headers may already be sent; nothing more we can do.
		return
	}
	if !found {
		http.NotFound(w, r)
	}
}

// --- GET /api/admin/apps ---

func (s *Server) handleListAdminApps(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status") // "", "pending", "approved", "rejected"

	var (
		regs []store.AppRegistration
		err  error
	)
	if statusFilter == "pending" {
		regs, err = s.store.ListPendingAppRegistrations(r.Context())
	} else {
		regs, err = s.store.ListAllAppRegistrations(r.Context())
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "list admin apps failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	apps := make([]appResponse, 0, len(regs))
	for _, reg := range regs {
		reg := reg
		client, err := s.pid.GetOIDCClient(r.Context(), reg.OIDCClientID)
		if err != nil {
			slog.WarnContext(r.Context(), "admin list: get oidc client failed", "client_id", reg.OIDCClientID, "err", err)
			continue
		}
		resp := buildAppResponse(client, &reg, "")
		// Populate owner username for admin context.
		if owner, err := s.pid.GetUser(r.Context(), reg.OwnerUserID); err == nil {
			resp.OwnerUsername = owner.Username
		}
		apps = append(apps, resp)
	}

	writeJSON(w, http.StatusOK, apps)
}

// --- POST /api/admin/apps/{id}/approve ---

func (s *Server) handleApproveApp(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	reg, err := s.store.GetAppRegistrationByClientID(r.Context(), clientID)
	if err == store.ErrNotFound {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "get app registration failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if reg.Status == "approved" {
		writeError(w, http.StatusConflict, "application is already approved")
		return
	}

	previousStatus := reg.Status
	previousReason := reg.RejectionReason
	if err := s.store.UpdateAppRegistrationStatus(r.Context(), clientID, "approved", ""); err != nil {
		slog.ErrorContext(r.Context(), "approve: update status failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := s.syncOIDCClientAccess(r, clientID, "approved", reg.VerifiedOnly); err != nil {
		if revertErr := s.store.UpdateAppRegistrationStatus(r.Context(), clientID, previousStatus, previousReason); revertErr != nil {
			slog.ErrorContext(r.Context(), "approve: revert status failed", "client_id", clientID, "err", revertErr)
		}
		slog.ErrorContext(r.Context(), "approve: sync client access failed", "client_id", clientID, "verified_only", reg.VerifiedOnly, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	client, err := s.pid.GetOIDCClient(r.Context(), clientID)
	if err != nil {
		slog.ErrorContext(r.Context(), "approve: get oidc client failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user, err := s.pid.GetUser(r.Context(), reg.OwnerUserID); err != nil {
		slog.WarnContext(r.Context(), "approve: get owner failed", "client_id", clientID, "owner_user_id", reg.OwnerUserID, "err", err)
	} else {
		s.mailer.SendAppApprovedNotification(user.Email, mailer.AppDecisionNotification{
			AppName:     client.Name,
			AppID:       client.ID,
			OwnerHandle: user.Username,
			ActionURL:   "https://" + r.Host + "/apps/" + client.ID,
			ActionLabel: "Open your application",
		})
	}
	reg.Status = "approved"
	reg.RejectionReason = ""
	writeJSON(w, http.StatusOK, buildAppResponse(client, reg, ""))
}

// --- POST /api/admin/apps/{id}/reject ---

type rejectRequest struct {
	Reason string `json:"reason"`
}

func (s *Server) handleRejectApp(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	reg, err := s.store.GetAppRegistrationByClientID(r.Context(), clientID)
	if err == store.ErrNotFound {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "get app registration failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var req rejectRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // reason is optional
	reason := strings.TrimSpace(req.Reason)

	// Ensure the client remains restricted.
	if err := s.syncOIDCClientAccess(r, clientID, "rejected", reg.VerifiedOnly); err != nil {
		slog.WarnContext(r.Context(), "reject: re-apply pending groups failed", "client_id", clientID, "err", err)
		// Non-fatal: continue with rejection.
	}

	if err := s.store.UpdateAppRegistrationStatus(r.Context(), clientID, "rejected", strings.TrimSpace(req.Reason)); err != nil {
		slog.ErrorContext(r.Context(), "reject: update status failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	client, err := s.pid.GetOIDCClient(r.Context(), clientID)
	if err != nil {
		slog.ErrorContext(r.Context(), "reject: get oidc client failed", "client_id", clientID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user, err := s.pid.GetUser(r.Context(), reg.OwnerUserID); err != nil {
		slog.WarnContext(r.Context(), "reject: get owner failed", "client_id", clientID, "owner_user_id", reg.OwnerUserID, "err", err)
	} else {
		s.mailer.SendAppRejectedNotification(user.Email, mailer.AppDecisionNotification{
			AppName:     client.Name,
			AppID:       client.ID,
			OwnerHandle: user.Username,
			Reason:      reason,
			ActionURL:   "https://" + r.Host + "/apps/" + client.ID,
			ActionLabel: "Open your application",
		})
	}
	reg.Status = "rejected"
	reg.RejectionReason = reason
	writeJSON(w, http.StatusOK, buildAppResponse(client, reg, ""))
}

// --- GET /api/apps/directory (public) ---

// directoryAppResponse is the public view of an app in the SCID directory.
type directoryAppResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	LaunchURL    string `json:"launch_url"`
	HasLogo      bool   `json:"has_logo"`
	VerifiedOnly bool   `json:"verified_only"`
	Category     string `json:"category,omitempty"`
	CreatedAt    string `json:"created_at"`
}

func (s *Server) handleListDirectoryApps(w http.ResponseWriter, r *http.Request) {
	regs, err := s.store.ListListedApps(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list directory apps failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	apps := make([]directoryAppResponse, 0, len(regs))
	for _, reg := range regs {
		reg := reg
		client, err := s.pid.GetOIDCClient(r.Context(), reg.OIDCClientID)
		if err != nil {
			slog.WarnContext(r.Context(), "directory: get oidc client failed", "client_id", reg.OIDCClientID, "err", err)
			continue
		}
		launchURL := ""
		if client.LaunchURL != nil {
			launchURL = *client.LaunchURL
		}
		if launchURL == "" {
			continue // skip apps without a launch URL
		}
		apps = append(apps, directoryAppResponse{
			ID:           client.ID,
			Name:         client.Name,
			Description:  reg.Description,
			LaunchURL:    launchURL,
			HasLogo:      client.HasLogo,
			VerifiedOnly: reg.VerifiedOnly,
			Category:     reg.Category,
			CreatedAt:    reg.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, apps)
}
