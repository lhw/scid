package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lhw/scid/companion/internal/pocketid"
)

var errMissingAuth = errors.New("missing auth")

type authCallbackRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
}

type authCallbackResponse struct {
	ReturnPath string `json:"return_path"`
}

type accessTokenSource int

const (
	accessTokenSourceNone accessTokenSource = iota
	accessTokenSourceBearer
	accessTokenSourceSession
)

func isValidAuthRedirectURL(raw string) bool {
	if !isValidCallbackURL(raw) {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Path == "/callback"
}

func sessionDeadline(now time.Time, sessionTTL time.Duration, tokenExpiry time.Time) time.Time {
	deadline := now.Add(sessionTTL)
	if !tokenExpiry.IsZero() && tokenExpiry.Before(deadline) {
		return tokenExpiry
	}
	return deadline
}

func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	var req authCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.CodeVerifier) == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code or verifier")
		return
	}
	req.RedirectURI = strings.TrimSpace(req.RedirectURI)
	if !isValidAuthRedirectURL(req.RedirectURI) {
		writeError(w, http.StatusBadRequest, "invalid redirect_uri")
		return
	}

	token, err := s.auth.ExchangeAuthorizationCode(r.Context(), req.RedirectURI, req.Code, req.CodeVerifier)
	if err != nil {
		slog.WarnContext(r.Context(), "auth callback token exchange failed", "err", err)
		writeError(w, http.StatusUnauthorized, "authentication failed")
		return
	}
	if token.AccessToken == "" {
		slog.WarnContext(r.Context(), "auth callback missing access token")
		writeError(w, http.StatusUnauthorized, "authentication failed")
		return
	}

	if err := s.sessions.RenewToken(r.Context()); err != nil {
		slog.ErrorContext(r.Context(), "renew session failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.sessions.Put(r.Context(), sessionAccessTokenKey, token.AccessToken)
	s.sessions.SetDeadline(r.Context(), sessionDeadline(time.Now(), s.cfg.SessionTTL, token.Expiry))

	writeJSON(w, http.StatusOK, authCallbackResponse{ReturnPath: "/"})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionFromRequest(r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) accessTokenFromRequest(r *http.Request) (string, accessTokenSource) {
	if token := extractBearerToken(r); token != "" {
		return token, accessTokenSourceBearer
	}

	accessToken := s.sessions.GetString(r.Context(), sessionAccessTokenKey)
	if accessToken == "" {
		return "", accessTokenSourceNone
	}
	return accessToken, accessTokenSourceSession
}

func (s *Server) clearSessionFromRequest(r *http.Request) {
	if err := s.sessions.Destroy(r.Context()); err != nil {
		slog.WarnContext(r.Context(), "destroy session failed", "err", err)
	}
}

func (s *Server) resolveAuthenticatedUser(r *http.Request) (*pocketid.User, error) {
	accessToken, source := s.accessTokenFromRequest(r)
	if source == accessTokenSourceNone {
		return nil, errMissingAuth
	}

	user, err := s.auth.GetCurrentUser(r.Context(), accessToken)
	if err != nil {
		if source == accessTokenSourceSession {
			s.clearSessionFromRequest(r)
		}
		return nil, err
	}
	return user, nil
}
