package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var turnstileHTTPClient = &http.Client{Timeout: 5 * time.Second}

type signupTokenRequest struct {
	TurnstileToken string `json:"turnstile_token"`
}

type signupTokenResponse struct {
	Token string `json:"token"`
}

// handleSignupToken validates a Cloudflare Turnstile captcha response, then
// creates a single-use Pocket ID signup token and returns it so the frontend
// can redirect the user to Pocket ID's registration page.
//
// POST /api/auth/signup-token  (public — no auth required)
func (s *Server) handleSignupToken(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	var req signupTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if s.cfg.TurnstileSecretKey != "" {
		clientIP := r.Header.Get("CF-Connecting-IP")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		if err := validateTurnstile(r.Context(), s.cfg.TurnstileSecretKey, req.TurnstileToken, clientIP); err != nil {
			slog.WarnContext(r.Context(), "turnstile validation failed", "err", err)
			writeError(w, http.StatusForbidden, "captcha validation failed")
			return
		}
	}

	st, err := s.pid.CreateSignupToken(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "create signup token failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create signup token")
		return
	}

	metricSignupTokens.Inc()
	auditLog(r.Context(), "signup_token.issued", "remote_ip", r.RemoteAddr)

	writeJSON(w, http.StatusCreated, signupTokenResponse{Token: st.Token})
}

// validateTurnstile calls Cloudflare's siteverify endpoint to confirm that the
// client-side Turnstile widget produced a genuine challenge response.
func validateTurnstile(ctx context.Context, secret, token, remoteIP string) error {
	if token == "" {
		return fmt.Errorf("missing turnstile token")
	}

	form := url.Values{
		"secret":   {secret},
		"response": {token},
		"remoteip": {remoteIP},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := turnstileHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Success bool     `json:"success"`
		Errors  []string `json:"error-codes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("turnstile rejected: %v", result.Errors)
	}
	return nil
}
