package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lhw/scid/companion/internal/rsi"
	"github.com/lhw/scid/companion/internal/store"
)

const (
	// orgLogoCacheDir is the shared companion data directory used for cached org logos.
	orgLogoDir    = "/data/org-logos"
	orgCacheTTL   = 7 * 24 * time.Hour // re-fetch org info after 1 week
	orgFetchAgent = "SCID-Companion/1.0 (Star Citizen Identity Provider; unofficial fansite tool)"
)

// orgHTTPClient is a dedicated client for fetching org logo images.
var orgHTTPClient = &http.Client{Timeout: 15 * time.Second}

// syncUserOrgs scrapes the user's RSI orgs page, caches org metadata and logos,
// and stores the user's org memberships in the DB. It is designed to be called
// non-fatally: errors are logged but do not fail the parent operation.
func (s *Server) syncUserOrgs(ctx context.Context, userID, handle string) {
	orgs, err := s.scraper.FetchOrgs(ctx, handle)
	if err != nil {
		slog.WarnContext(ctx, "sync orgs: fetch failed", "handle", handle, "err", err)
		return
	}

	// Ensure the logo cache directory exists.
	if err := os.MkdirAll(orgLogoDir, 0755); err != nil {
		slog.WarnContext(ctx, "sync orgs: mkdir logo dir", "err", err)
		// Continue — we can still cache the names.
	}

	userOrgs := make([]store.UserOrg, 0, len(orgs))
	for _, org := range orgs {
		if org.SID == "" {
			continue
		}

		// Cache org metadata (name + logo) if stale or missing.
		cached, _ := s.store.GetOrgCache(ctx, org.SID)
		if cached == nil || time.Since(cached.FetchedAt) > orgCacheTTL || (cached.LogoPath == "" && org.LogoURL != "") {
			logoPath := ""
			if org.LogoURL != "" {
				logoPath = cacheOrgLogo(ctx, org.SID, org.LogoURL)
			}
			name := org.Name
			if name == "" && cached != nil {
				name = cached.Name // keep existing name if scraper didn't find one
			}
			if err := s.store.UpsertOrgCache(ctx, &store.OrgCacheEntry{
				SID:       org.SID,
				Name:      name,
				LogoPath:  logoPath,
				FetchedAt: time.Now().UTC(),
			}); err != nil {
				slog.WarnContext(ctx, "sync orgs: upsert cache", "sid", org.SID, "err", err)
			}
		}

		userOrgs = append(userOrgs, store.UserOrg{
			PocketIDUserID: userID,
			SID:            org.SID,
			RankName:       org.RankName,
			IsMain:         org.IsMain,
		})
	}

	if err := s.store.SetUserOrgs(ctx, userID, userOrgs); err != nil {
		slog.WarnContext(ctx, "sync orgs: set user orgs", "user_id", userID, "err", err)
	}

	// Sync rsi:<SID> groups to Pocket ID so OIDC tokens carry org membership.
	s.syncOrgGroupsToPocketID(ctx, userID, orgs)
}

// syncOrgGroupsToPocketID ensures that Pocket ID has an rsi:<SID> group for
// every org and that the user's group membership reflects their current orgs.
// It keeps all non-rsi: groups the user already has; only rsi: groups are
// managed here.  Errors are logged non-fatally.
func (s *Server) syncOrgGroupsToPocketID(ctx context.Context, userID string, orgs []rsi.OrgInfo) {
	// Get the user's current groups so we can preserve non-rsi ones.
	currentGroups, err := s.pid.GetUserGroups(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "sync org groups: get user groups", "user_id", userID, "err", err)
		return
	}

	// Collect group IDs to keep (everything that is NOT an rsi: group).
	keepIDs := make([]string, 0, len(currentGroups))
	for _, g := range currentGroups {
		if !strings.HasPrefix(g.Name, "rsi:") {
			keepIDs = append(keepIDs, g.ID)
		}
	}

	// For each scraped org, ensure the rsi:<SID> group exists and collect its ID.
	newOrgGroupIDs := make([]string, 0, len(orgs))
	for _, org := range orgs {
		if org.SID == "" {
			continue
		}
		groupName := "rsi:" + strings.ToUpper(org.SID)
		friendlyName := org.Name
		if friendlyName == "" {
			friendlyName = groupName
		}
		g, err := s.pid.EnsureGroupExists(ctx, groupName, friendlyName)
		if err != nil {
			slog.WarnContext(ctx, "sync org groups: ensure group", "group", groupName, "err", err)
			continue
		}
		newOrgGroupIDs = append(newOrgGroupIDs, g.ID)
	}

	// Merge: preserved non-rsi groups + new rsi groups.
	merged := append(keepIDs, newOrgGroupIDs...)

	if err := s.pid.SetUserGroups(ctx, userID, merged); err != nil {
		slog.WarnContext(ctx, "sync org groups: set user groups", "user_id", userID, "err", err)
	}
}

// cacheOrgLogo downloads logoURL and saves it to orgLogoDir/<SID>.png.
// Returns the saved path, or "" on any error (non-fatal).
func cacheOrgLogo(ctx context.Context, sid, logoURL string) string {
	if logoURL == "" {
		return ""
	}

	if !rsi.IsAllowedImageURL(logoURL) {
		slog.WarnContext(ctx, "cache org logo: untrusted URL", "sid", sid, "url", logoURL)
		return ""
	}

	// Determine extension from URL (default to .png).
	ext := ".png"
	if idx := strings.LastIndex(logoURL, "."); idx != -1 {
		candidate := strings.ToLower(logoURL[idx:])
		// Strip query strings
		if q := strings.Index(candidate, "?"); q != -1 {
			candidate = candidate[:q]
		}
		if candidate == ".jpg" || candidate == ".jpeg" || candidate == ".png" || candidate == ".webp" {
			ext = candidate
			if ext == ".jpeg" {
				ext = ".jpg"
			}
		}
	}

	destPath := filepath.Join(orgLogoDir, strings.ToUpper(sid)+ext)

	fetchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, logoURL, nil)
	if err != nil {
		slog.WarnContext(ctx, "cache org logo: build request", "sid", sid, "err", err)
		return ""
	}
	req.Header.Set("User-Agent", orgFetchAgent)

	resp, err := orgHTTPClient.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "cache org logo: fetch", "sid", sid, "err", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.WarnContext(ctx, "cache org logo: non-200", "sid", sid, "status", resp.StatusCode)
		return ""
	}

	f, err := os.CreateTemp(orgLogoDir, "org-logo-tmp-*")
	if err != nil {
		slog.WarnContext(ctx, "cache org logo: create temp", "sid", sid, "err", err)
		return ""
	}
	tmpName := f.Name()
	defer func() {
		if tmpName != "" {
			os.Remove(tmpName) // clean up on failure
		}
	}()

	if _, err := io.Copy(f, io.LimitReader(resp.Body, 2<<20 /* 2 MiB max */)); err != nil {
		f.Close()
		slog.WarnContext(ctx, "cache org logo: write", "sid", sid, "err", err)
		return ""
	}
	f.Close()

	if err := os.Rename(tmpName, destPath); err != nil {
		slog.WarnContext(ctx, "cache org logo: rename", "sid", sid, "err", err)
		return ""
	}
	tmpName = "" // prevent deferred cleanup
	return destPath
}

// handleOrgLogo serves the cached logo for an org SID from the filesystem.
// GET /api/orgs/{sid}/logo
func (s *Server) handleOrgLogo(w http.ResponseWriter, r *http.Request) {
	sid := strings.ToUpper(chi.URLParam(r, "sid"))
	if !isValidSID(sid) {
		http.NotFound(w, r)
		return
	}

	cached, err := s.store.GetOrgCache(r.Context(), sid)
	if err != nil || cached == nil || cached.LogoPath == "" {
		http.NotFound(w, r)
		return
	}

	// Ensure the path is strictly within the logo directory (prevent path traversal).
	abs, err := filepath.Abs(cached.LogoPath)
	if err != nil || !strings.HasPrefix(abs, orgLogoDir+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, abs)
}

// isValidSID checks that a SID is safe to use as a DB key and filename.
func isValidSID(sid string) bool {
	if len(sid) < 1 || len(sid) > 16 {
		return false
	}
	for _, c := range sid {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// OrgEntry is the JSON shape returned in statusResponse.Orgs.
type OrgEntry struct {
	SID      string `json:"sid"`
	Name     string `json:"name,omitempty"`
	RankName string `json:"rank_name,omitempty"`
	IsMain   bool   `json:"is_main,omitempty"`
	HasLogo  bool   `json:"has_logo,omitempty"`
}
