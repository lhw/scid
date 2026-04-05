package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lhw/scid/companion/internal/store"
)

// ---- Admin: User management ----

type adminUserEntry struct {
	UserID        string `json:"user_id"`
	Handle        string `json:"handle"`
	VerifiedAt    string `json:"verified_at"`
	HandleBlocked bool   `json:"handle_blocked"`
}

// handleListAdminUsers returns all verified users with their block status.
// GET /api/admin/users
func (s *Server) handleListAdminUsers(w http.ResponseWriter, r *http.Request) {
	handles, err := s.store.ListVerifiedHandles(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list verified handles", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	blocked, err := s.store.ListBlockedHandles(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list blocked handles", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	blockedSet := make(map[string]bool, len(blocked))
	for _, b := range blocked {
		blockedSet[strings.ToLower(b.RSIHandle)] = true
	}

	entries := make([]adminUserEntry, 0, len(handles))
	for _, h := range handles {
		entries = append(entries, adminUserEntry{
			UserID:        h.PocketIDUserID,
			Handle:        h.RSIHandle,
			VerifiedAt:    h.VerifiedAt.UTC().Format("2006-01-02T15:04:05Z"),
			HandleBlocked: blockedSet[strings.ToLower(h.RSIHandle)],
		})
	}
	writeJSON(w, http.StatusOK, entries)
}

type adminDeleteUserRequest struct {
	BlockHandle bool   `json:"block_handle"`
	Reason      string `json:"reason"`
}

// handleDeleteAdminUser removes a user from Pocket ID and cleans up SCID data.
// Optionally blocks the user's RSI handle so they cannot re-verify.
// DELETE /api/admin/users/{id}
func (s *Server) handleDeleteAdminUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "missing user id")
		return
	}

	var req adminDeleteUserRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Body is optional — ignore parse errors
		req = adminDeleteUserRequest{}
	}

	admin := userFromContext(r.Context())
	adminName := ""
	if admin != nil {
		adminName = admin.Username
	}

	// Look up RSI handle before deleting so we can block it if requested.
	var handleToBlock string
	if req.BlockHandle {
		entry, err := s.store.GetVerifiedHandleByUserID(r.Context(), userID)
		if err == nil && entry != nil {
			handleToBlock = entry.RSIHandle
		}
	}

	// Delete from SCID local store first (cascades verification data).
	if err := s.store.DeleteUserData(r.Context(), userID); err != nil {
		slog.ErrorContext(r.Context(), "admin delete user: local data", "user_id", userID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Delete from Pocket ID.
	if err := s.pid.DeleteUser(r.Context(), userID); err != nil {
		slog.WarnContext(r.Context(), "admin delete user: pocket id", "user_id", userID, "err", err)
		// Non-fatal — the local cleanup already happened.
	}

	// Optionally block the handle.
	if handleToBlock != "" {
		reason := req.Reason
		if reason == "" {
			reason = "removed by admin"
		}
		if err := s.store.BlockHandle(r.Context(), handleToBlock, adminName, reason); err != nil {
			slog.WarnContext(r.Context(), "admin delete user: block handle", "handle", handleToBlock, "err", err)
		}
	}

	auditLog(r.Context(), "admin.user.deleted",
		"user_id", userID,
		"block_handle", req.BlockHandle,
		"admin", adminName)

	w.WriteHeader(http.StatusNoContent)
}

type adminBlockHandleRequest struct {
	Handle string `json:"handle"`
	Reason string `json:"reason"`
}

// handleBlockHandle adds an RSI handle to the blocklist.
// POST /api/admin/handles/block
func (s *Server) handleBlockHandle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	var req adminBlockHandleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Handle = strings.TrimSpace(req.Handle)
	if !handleRegexp.MatchString(req.Handle) {
		writeError(w, http.StatusBadRequest, "invalid RSI handle format")
		return
	}

	admin := userFromContext(r.Context())
	adminName := ""
	if admin != nil {
		adminName = admin.Username
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "blocked by admin"
	}

	if err := s.store.BlockHandle(r.Context(), req.Handle, adminName, reason); err != nil {
		slog.ErrorContext(r.Context(), "block handle", "handle", req.Handle, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	auditLog(r.Context(), "admin.handle.blocked", "handle", req.Handle, "admin", adminName)
	w.WriteHeader(http.StatusNoContent)
}

// handleUnblockHandle removes an RSI handle from the blocklist.
// DELETE /api/admin/handles/{handle}
func (s *Server) handleUnblockHandle(w http.ResponseWriter, r *http.Request) {
	handle := chi.URLParam(r, "handle")
	if handle == "" {
		writeError(w, http.StatusBadRequest, "missing handle")
		return
	}

	if err := s.store.UnblockHandle(r.Context(), handle); err != nil {
		slog.ErrorContext(r.Context(), "unblock handle", "handle", handle, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	admin := userFromContext(r.Context())
	adminName := ""
	if admin != nil {
		adminName = admin.Username
	}
	auditLog(r.Context(), "admin.handle.unblocked", "handle", handle, "admin", adminName)
	w.WriteHeader(http.StatusNoContent)
}

// handleListBlockedHandles returns all blocked handles.
// GET /api/admin/handles/blocked
func (s *Server) handleListBlockedHandles(w http.ResponseWriter, r *http.Request) {
	blocked, err := s.store.ListBlockedHandles(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list blocked handles", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	type blockedEntry struct {
		Handle    string `json:"handle"`
		BlockedAt string `json:"blocked_at"`
		BlockedBy string `json:"blocked_by"`
		Reason    string `json:"reason"`
	}
	entries := make([]blockedEntry, 0, len(blocked))
	for _, b := range blocked {
		entries = append(entries, blockedEntry{
			Handle:    b.RSIHandle,
			BlockedAt: b.BlockedAt.UTC().Format("2006-01-02T15:04:05Z"),
			BlockedBy: b.BlockedBy,
			Reason:    b.Reason,
		})
	}
	writeJSON(w, http.StatusOK, entries)
}

// ---- Admin: Org logo management ----

type adminOrgEntry struct {
	SID         string `json:"sid"`
	Name        string `json:"name"`
	HasLogo     bool   `json:"has_logo"`
	LogoBlocked bool   `json:"logo_blocked"`
	FetchedAt   string `json:"fetched_at"`
}

// handleListAdminOrgs returns all cached org entries.
// GET /api/admin/orgs
func (s *Server) handleListAdminOrgs(w http.ResponseWriter, r *http.Request) {
	orgs, err := s.store.ListOrgCache(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list org cache", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	entries := make([]adminOrgEntry, 0, len(orgs))
	for _, o := range orgs {
		entries = append(entries, adminOrgEntry{
			SID:         o.SID,
			Name:        o.Name,
			HasLogo:     o.LogoPath != "",
			LogoBlocked: o.LogoBlocked,
			FetchedAt:   o.FetchedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	writeJSON(w, http.StatusOK, entries)
}

// handleBlockOrgLogo sets logo_blocked=1 for an org and deletes the cached file.
// POST /api/admin/orgs/{sid}/block-logo
func (s *Server) handleBlockOrgLogo(w http.ResponseWriter, r *http.Request) {
	sid := strings.ToUpper(chi.URLParam(r, "sid"))
	if !isValidSID(sid) {
		writeError(w, http.StatusBadRequest, "invalid SID")
		return
	}

	cached, err := s.store.GetOrgCache(r.Context(), sid)
	if err == store.ErrNotFound {
		writeError(w, http.StatusNotFound, "org not found in cache")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Delete the cached logo file from disk.
	if cached.LogoPath != "" {
		if removeErr := os.Remove(cached.LogoPath); removeErr != nil && !os.IsNotExist(removeErr) {
			slog.WarnContext(r.Context(), "block org logo: remove file", "sid", sid, "err", removeErr)
		}
	}

	// Clear the logo path and set blocked flag.
	if err := s.store.UpsertOrgCache(r.Context(), &store.OrgCacheEntry{
		SID:       sid,
		Name:      cached.Name,
		LogoPath:  "",
		FetchedAt: cached.FetchedAt,
	}); err != nil {
		slog.ErrorContext(r.Context(), "block org logo: clear logo path", "sid", sid, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := s.store.SetOrgLogoBlocked(r.Context(), sid, true); err != nil {
		slog.ErrorContext(r.Context(), "block org logo: set blocked", "sid", sid, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	admin := userFromContext(r.Context())
	adminName := ""
	if admin != nil {
		adminName = admin.Username
	}
	auditLog(r.Context(), "admin.org.logo.blocked", "sid", sid, "admin", adminName)
	w.WriteHeader(http.StatusNoContent)
}

// handleUnblockOrgLogo clears the logo_blocked flag for an org.
// DELETE /api/admin/orgs/{sid}/block-logo
func (s *Server) handleUnblockOrgLogo(w http.ResponseWriter, r *http.Request) {
	sid := strings.ToUpper(chi.URLParam(r, "sid"))
	if !isValidSID(sid) {
		writeError(w, http.StatusBadRequest, "invalid SID")
		return
	}

	if err := s.store.SetOrgLogoBlocked(r.Context(), sid, false); err != nil {
		slog.ErrorContext(r.Context(), "unblock org logo", "sid", sid, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Re-download the logo immediately using the stored URL so it reappears
	// without waiting for the next user-triggered sync. If the URL isn't
	// stored yet (older cache entries), fall back to marking the entry stale
	// so the next syncUserOrgs pass re-fetches it.
	if cached, err := s.store.GetOrgCache(r.Context(), sid); err == nil {
		newLogoPath := ""
		if cached.LogoURL != "" {
			newLogoPath = cacheOrgLogo(r.Context(), sid, cached.LogoURL)
		}
		_ = s.store.UpsertOrgCache(r.Context(), &store.OrgCacheEntry{
			SID:       sid,
			Name:      cached.Name,
			LogoPath:  newLogoPath,
			LogoURL:   cached.LogoURL,
			FetchedAt: time.Time{},
		})
	}

	admin := userFromContext(r.Context())
	adminName := ""
	if admin != nil {
		adminName = admin.Username
	}
	auditLog(r.Context(), "admin.org.logo.unblocked", "sid", sid, "admin", adminName)
	w.WriteHeader(http.StatusNoContent)
}

// ---- Admin: Report review queue ----

type adminReportEntry struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Target     string `json:"target"`
	Reason     string `json:"reason"`
	ReporterIP string `json:"reporter_ip"`
	CreatedAt  string `json:"created_at"`
	Status     string `json:"status"`
	ReviewedBy string `json:"reviewed_by,omitempty"`
	ReviewedAt string `json:"reviewed_at,omitempty"`
}

func reportToEntry(r store.Report) adminReportEntry {
	e := adminReportEntry{
		ID:         r.ID,
		Type:       r.Type,
		Target:     r.Target,
		Reason:     r.Reason,
		ReporterIP: r.ReporterIP,
		CreatedAt:  r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Status:     r.Status,
		ReviewedBy: r.ReviewedBy,
	}
	if !r.ReviewedAt.IsZero() {
		e.ReviewedAt = r.ReviewedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return e
}

// handleListAdminReports returns reports, optionally filtered by status.
// GET /api/admin/reports?status=pending
func (s *Server) handleListAdminReports(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	reports, err := s.store.ListReports(r.Context(), status)
	if err != nil {
		slog.ErrorContext(r.Context(), "list reports", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	entries := make([]adminReportEntry, 0, len(reports))
	for _, rpt := range reports {
		entries = append(entries, reportToEntry(rpt))
	}
	writeJSON(w, http.StatusOK, entries)
}

// handleReviewReport marks a report as reviewed (admin took action).
// POST /api/admin/reports/{id}/review
func (s *Server) handleReviewReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	admin := userFromContext(r.Context())
	adminName := ""
	if admin != nil {
		adminName = admin.Username
	}
	if err := s.store.UpdateReportStatus(r.Context(), id, "reviewed", adminName); err != nil {
		slog.ErrorContext(r.Context(), "review report", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleDismissReport marks a report as dismissed (no action needed).
// POST /api/admin/reports/{id}/dismiss
func (s *Server) handleDismissReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	admin := userFromContext(r.Context())
	adminName := ""
	if admin != nil {
		adminName = admin.Username
	}
	if err := s.store.UpdateReportStatus(r.Context(), id, "dismissed", adminName); err != nil {
		slog.ErrorContext(r.Context(), "dismiss report", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
