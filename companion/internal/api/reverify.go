package api

import (
	"context"
	"log/slog"
	"time"
)

const (
	orgReverifyAge    = 7 * 24 * time.Hour     // re-sync orgs 7 days after last sync
	reverifyJobPoll   = 1 * time.Hour          // how often the job checks for due users
	reverifyUserDelay = 10 * time.Second       // polite delay between RSI scrape calls
	reverifySeedDelay = 100 * time.Millisecond // delay between Pocket ID calls during seeding
)

// StartReverifyJob launches the background org re-sync goroutine.
//
// On first start it seeds the sync schedule for any existing verified users
// by inserting a user_org_sync row (INSERT OR IGNORE) with synced_at set to
// their rsi_verified_at claim. This staggers the initial wave of re-syncs
// naturally — a user verified 3 days ago won't be due for another 4 days;
// one verified 10 days ago will be processed in the very first pass.
//
// After seeding, the job polls hourly and calls syncUserOrgs (from orgs.go)
// for every user whose 7-day timer has lapsed, spacing requests 10 s apart
// to avoid triggering RSI's crawler defences.
func (s *Server) StartReverifyJob(ctx context.Context) {
	go func() {
		// Brief startup delay so the service is fully up before making API calls.
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Minute):
		}

		// Enroll existing verified users that don't yet have a sync entry.
		s.seedOrgSyncTable(ctx)

		// Process any users that are immediately due before the first tick.
		s.runOrgSyncPass(ctx)

		ticker := time.NewTicker(reverifyJobPoll)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				slog.Info("reverify job: shutting down")
				return
			case <-ticker.C:
				s.runOrgSyncPass(ctx)
			}
		}
	}()
}

// seedOrgSyncTable enumerates the "verified" Pocket ID group and inserts a
// row in user_org_sync for every member that doesn't already have one. The
// INSERT OR IGNORE leaves existing schedules undisturbed.
func (s *Server) seedOrgSyncTable(ctx context.Context) {
	members, err := s.pid.ListGroupMembers(ctx, "verified")
	if err != nil {
		slog.Error("reverify: seed: list group members", "err", err)
		return
	}
	slog.Info("reverify: seeding org sync table", "members", len(members))

	for _, u := range members {
		select {
		case <-ctx.Done():
			return
		default:
		}

		detail, err := s.pid.GetUser(ctx, u.ID)
		if err != nil {
			slog.Warn("reverify: seed: get user", "user_id", u.ID, "err", err)
			time.Sleep(reverifySeedDelay)
			continue
		}

		var handle, verifiedAtStr string
		for _, c := range detail.CustomClaims {
			switch c.Key {
			case "rsi_handle":
				handle = c.Value
			case "rsi_verified_at":
				verifiedAtStr = c.Value
			}
		}
		if handle == "" {
			time.Sleep(reverifySeedDelay)
			continue
		}

		// Seed synced_at to the user's original verified_at so their 7-day
		// expiry window is relative to when they actually verified.
		syncedAt, err := time.Parse(time.RFC3339, verifiedAtStr)
		if err != nil {
			syncedAt = time.Time{} // epoch — immediately due
		}
		if err := s.store.InsertOrgSyncIfMissing(ctx, u.ID, handle, syncedAt); err != nil {
			slog.Warn("reverify: seed: insert entry", "user_id", u.ID, "err", err)
		}

		time.Sleep(reverifySeedDelay)
	}
	slog.Info("reverify: seed complete", "total", len(members))
}

// runOrgSyncPass queries user_org_sync for users whose sync has expired and
// refreshes each one's RSI org memberships with a polite delay between calls.
func (s *Server) runOrgSyncPass(ctx context.Context) {
	due, err := s.store.ListExpiredOrgSyncs(ctx, time.Now().Add(-orgReverifyAge))
	if err != nil {
		slog.Error("reverify: list expired org syncs", "err", err)
		return
	}
	if len(due) == 0 {
		return
	}

	slog.Info("reverify: syncing orgs for due users", "count", len(due))

	for _, entry := range due {
		select {
		case <-ctx.Done():
			return
		default:
		}

		slog.Debug("reverify: syncing user orgs", "user_id", entry.PocketIDUserID, "handle", entry.Handle)
		s.syncUserOrgs(ctx, entry.PocketIDUserID, entry.Handle)
		metricOrgSyncs.WithLabelValues("synced").Inc()

		// Stamp the sync time regardless of whether syncUserOrgs had errors —
		// it logs its own warnings and we don't want one failed user to repeat
		// on every hourly poll.
		if err := s.store.UpsertOrgSync(ctx, entry.PocketIDUserID, entry.Handle, time.Now()); err != nil {
			slog.Warn("reverify: update org sync timestamp", "user_id", entry.PocketIDUserID, "err", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(reverifyUserDelay):
		}
	}

	slog.Info("reverify: org sync pass complete", "count", len(due))
}
