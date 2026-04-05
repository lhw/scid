package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lhw/scid/companion/internal/store"
)

type submitReportRequest struct {
	Type           string `json:"type"`   // "user" | "org"
	Target         string `json:"target"` // RSI handle or org SID
	Reason         string `json:"reason"`
	TurnstileToken string `json:"turnstile_token"`
}

type submitReportResponse struct {
	ID string `json:"id"`
}

// handleSubmitReport accepts a public abuse report for a user handle or org SID.
// It validates the Cloudflare Turnstile token when one is configured, then stores
// the report for admin review.
//
// POST /api/report  (public — no login required)
func (s *Server) handleSubmitReport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var req submitReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Type = strings.TrimSpace(req.Type)
	req.Target = strings.TrimSpace(req.Target)
	req.Reason = strings.TrimSpace(req.Reason)

	if req.Type != "user" && req.Type != "org" {
		writeError(w, http.StatusBadRequest, "type must be 'user' or 'org'")
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	if req.Type == "user" && !handleRegexp.MatchString(req.Target) {
		writeError(w, http.StatusBadRequest, "invalid RSI handle format")
		return
	}
	if req.Type == "org" && !isValidSID(strings.ToUpper(req.Target)) {
		writeError(w, http.StatusBadRequest, "invalid org SID format")
		return
	}
	if len(req.Reason) < 10 {
		writeError(w, http.StatusBadRequest, "reason must be at least 10 characters")
		return
	}
	if len(req.Reason) > 2000 {
		writeError(w, http.StatusBadRequest, "reason too long (max 2000 characters)")
		return
	}

	// Validate Turnstile if the secret is configured.
	if s.cfg.TurnstileSecretKey != "" {
		clientIP := r.Header.Get("CF-Connecting-IP")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		if err := validateTurnstile(r.Context(), s.cfg.TurnstileSecretKey, req.TurnstileToken, clientIP); err != nil {
			slog.WarnContext(r.Context(), "report: turnstile failed", "err", err)
			writeError(w, http.StatusForbidden, "captcha validation failed")
			return
		}
	}

	id, err := newID()
	if err != nil {
		slog.ErrorContext(r.Context(), "report: generate id", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	reporterIP := r.Header.Get("CF-Connecting-IP")
	if reporterIP == "" {
		reporterIP = r.RemoteAddr
	}

	target := req.Target
	if req.Type == "org" {
		target = strings.ToUpper(target)
	}

	report := &store.Report{
		ID:         id,
		Type:       req.Type,
		Target:     target,
		Reason:     req.Reason,
		ReporterIP: reporterIP,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.store.CreateReport(r.Context(), report); err != nil {
		slog.ErrorContext(r.Context(), "report: create", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	slog.InfoContext(r.Context(), "report submitted", "id", id, "type", req.Type, "target", target)
	writeJSON(w, http.StatusCreated, submitReportResponse{ID: id})
}
