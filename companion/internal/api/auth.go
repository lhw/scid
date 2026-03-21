package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lhw/scid/companion/internal/pocketid"
)

var errMissingAuth = errors.New("missing auth")

type authCallbackRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
}

type authCallbackResponse struct {
	ReturnPath string `json:"return_path"`
}

func callbackURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host + "/callback"
}

func sessionDuration(sessionTTL time.Duration, expiresIn int) time.Duration {
	if expiresIn <= 0 {
		return sessionTTL
	}
	tokenTTL := time.Duration(expiresIn) * time.Second
	if tokenTTL < sessionTTL {
		return tokenTTL
	}
	return sessionTTL
}

func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	var req authCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.CodeVerifier) == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code or verifier")
		return
	}

	token, err := s.pid.ExchangeAuthorizationCode(
		r.Context(),
		s.cfg.OIDCClientID,
		callbackURL(r),
		req.Code,
		req.CodeVerifier,
	)
	if err != nil {
		slog.WarnContext(r.Context(), "auth callback token exchange failed", "err", err)
		writeError(w, http.StatusUnauthorized, "authentication failed")
		return
	}

	cookieValue, expiresAt, err := s.sessions.create(token.AccessToken, sessionDuration(s.cfg.SessionTTL, token.ExpiresIn))
	if err != nil {
		slog.ErrorContext(r.Context(), "create session failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	setSessionCookie(w, cookieValue, expiresAt)
	writeJSON(w, http.StatusOK, authCallbackResponse{ReturnPath: "/"})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionFromRequest(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) accessTokenFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	if token := extractBearerToken(r); token != "" {
		return token, true
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}
	accessToken, ok := s.sessions.lookup(cookie.Value)
	if !ok {
		clearSessionCookie(w)
		return "", false
	}
	return accessToken, true
}

func (s *Server) clearSessionFromRequest(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.delete(cookie.Value)
	}
	clearSessionCookie(w)
}

func (s *Server) resolveAuthenticatedUser(w http.ResponseWriter, r *http.Request) (*pocketid.User, error) {
	accessToken, ok := s.accessTokenFromRequest(w, r)
	if !ok {
		return nil, errMissingAuth
	}

	user, err := s.pid.GetCurrentUser(r.Context(), accessToken)
	if err != nil {
		s.clearSessionFromRequest(w, r)
		return nil, err
	}
	return user, nil
}