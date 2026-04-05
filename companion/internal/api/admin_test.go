package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lhw/scid/companion/internal/rsi"
	"github.com/lhw/scid/companion/internal/store"
)

// addAdmin registers a user in both the mock PocketID AND the "admin" group.
func addAdmin(env *testEnv, token, id, username string) {
	env.addUser(token, id, username, "admin")
}

// ────────────────────────────────────────────────────────────────────────────
// Auth guards
// ────────────────────────────────────────────────────────────────────────────

// TestAdminEndpoints_RequireAuth verifies that unauthenticated requests to every
// admin route are rejected with 401.
func TestAdminEndpoints_RequireAuth(t *testing.T) {
	env := newTestEnv(t, false)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/admin/users"},
		{http.MethodDelete, "/api/admin/users/some-id"},
		{http.MethodGet, "/api/admin/handles/blocked"},
		{http.MethodPost, "/api/admin/handles/block"},
		{http.MethodDelete, "/api/admin/handles/SomeHandle"},
		{http.MethodGet, "/api/admin/orgs"},
		{http.MethodPost, "/api/admin/orgs/SPAWO/block-logo"},
		{http.MethodDelete, "/api/admin/orgs/SPAWO/block-logo"},
		{http.MethodGet, "/api/admin/reports"},
		{http.MethodPost, "/api/admin/reports/some-id/review"},
		{http.MethodPost, "/api/admin/reports/some-id/dismiss"},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			resp := env.do(rt.method, rt.path, "", nil)
			env.mustStatus(resp, http.StatusUnauthorized)
			env.drain(resp)
		})
	}
}

// TestAdminEndpoints_RequireAdminGroup verifies that an authenticated but
// non-admin user is rejected with 403.
func TestAdminEndpoints_RequireAdminGroup(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice", "verified") // no "admin" group

	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/admin/users"},
		{http.MethodGet, "/api/admin/handles/blocked"},
		{http.MethodGet, "/api/admin/orgs"},
		{http.MethodGet, "/api/admin/reports"},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			resp := env.do(rt.method, rt.path, "tok-alice", nil)
			env.mustStatus(resp, http.StatusForbidden)
			env.drain(resp)
		})
	}
}

// ────────────────────────────────────────────────────────────────────────────
// User management
// ────────────────────────────────────────────────────────────────────────────

// TestAdminListUsers_Empty returns an empty array when no users are verified.
func TestAdminListUsers_Empty(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	resp := env.do(http.MethodGet, "/api/admin/users", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)

	var users []adminUserEntry
	env.decodeJSON(resp, &users)
	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}

// TestAdminListUsers_ShowsVerifiedUsers shows that verified users appear in the list.
func TestAdminListUsers_ShowsVerifiedUsers(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	// Verify alice through the normal flow.
	resp := env.do(http.MethodPost, "/api/verify/start", "tok-alice", map[string]string{"handle": "AliceRSI"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)
	env.scraper.setProfile(&rsi.Profile{Bio: sr.Token})
	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	resp = env.do(http.MethodGet, "/api/admin/users", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)

	var users []adminUserEntry
	env.decodeJSON(resp, &users)
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].Handle != "AliceRSI" {
		t.Errorf("expected handle AliceRSI, got %q", users[0].Handle)
	}
	if users[0].UserID != "user-alice" {
		t.Errorf("expected user_id user-alice, got %q", users[0].UserID)
	}
	if users[0].HandleBlocked {
		t.Error("handle should not be blocked")
	}
}

// TestAdminDeleteUser removes all local data for a user.
func TestAdminDeleteUser(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	// Verify alice.
	resp := env.do(http.MethodPost, "/api/verify/start", "tok-alice", map[string]string{"handle": "DeleteMe"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)
	env.scraper.setProfile(&rsi.Profile{Bio: sr.Token})
	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	// Delete the user.
	resp = env.do(http.MethodDelete, "/api/admin/users/user-alice", "tok-admin",
		map[string]any{"block_handle": false})
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// User should no longer appear in admin list.
	resp = env.do(http.MethodGet, "/api/admin/users", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var users []adminUserEntry
	env.decodeJSON(resp, &users)
	if len(users) != 0 {
		t.Fatalf("expected 0 users after delete, got %d", len(users))
	}
}

// TestAdminDeleteUser_WithHandleBlock blocks the RSI handle on deletion.
func TestAdminDeleteUser_WithHandleBlock(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")
	env.addUser("tok-bob", "user-bob", "bob", "verified")

	// Verify bob.
	resp := env.do(http.MethodPost, "/api/verify/start", "tok-bob", map[string]string{"handle": "BobHandle"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)
	env.scraper.setProfile(&rsi.Profile{Bio: sr.Token})
	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-bob", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	// Delete with block_handle=true.
	resp = env.do(http.MethodDelete, "/api/admin/users/user-bob", "tok-admin",
		map[string]any{"block_handle": true, "reason": "test block"})
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// Handle should now be in the blocked list.
	resp = env.do(http.MethodGet, "/api/admin/handles/blocked", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var blocked []struct {
		Handle string `json:"handle"`
		Reason string `json:"reason"`
	}
	env.decodeJSON(resp, &blocked)
	found := false
	for _, b := range blocked {
		if strings.EqualFold(b.Handle, "BobHandle") {
			found = true
		}
	}
	if !found {
		t.Error("BobHandle expected to be in blocked list after delete-with-block")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Handle blocklist
// ────────────────────────────────────────────────────────────────────────────

// TestAdminBlockUnblockHandle tests the full lifecycle of a handle block.
func TestAdminBlockUnblockHandle(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	// List should be empty initially.
	resp := env.do(http.MethodGet, "/api/admin/handles/blocked", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var blocked []struct{ Handle string }
	env.decodeJSON(resp, &blocked)
	if len(blocked) != 0 {
		t.Fatalf("expected empty blocked list, got %d", len(blocked))
	}

	// Block a handle.
	resp = env.do(http.MethodPost, "/api/admin/handles/block", "tok-admin",
		map[string]string{"handle": "BlockedUser", "reason": "spam"})
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// It should appear in the list.
	resp = env.do(http.MethodGet, "/api/admin/handles/blocked", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var blockedAfter []struct {
		Handle string `json:"handle"`
		Reason string `json:"reason"`
	}
	env.decodeJSON(resp, &blockedAfter)
	if len(blockedAfter) != 1 {
		t.Fatalf("expected 1 blocked handle, got %d", len(blockedAfter))
	}
	if blockedAfter[0].Handle != "BlockedUser" {
		t.Errorf("expected BlockedUser, got %q", blockedAfter[0].Handle)
	}
	if blockedAfter[0].Reason != "spam" {
		t.Errorf("expected reason spam, got %q", blockedAfter[0].Reason)
	}

	// Unblock.
	resp = env.do(http.MethodDelete, "/api/admin/handles/BlockedUser", "tok-admin", nil)
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// Should be gone.
	resp = env.do(http.MethodGet, "/api/admin/handles/blocked", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var blockedFinal []struct{ Handle string }
	env.decodeJSON(resp, &blockedFinal)
	if len(blockedFinal) != 0 {
		t.Fatalf("expected empty list after unblock, got %d", len(blockedFinal))
	}
}

// TestAdminBlockHandle_InvalidFormat rejects handles that don't match the pattern.
func TestAdminBlockHandle_InvalidFormat(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	cases := []string{"ab", "bad handle!", "x" + strings.Repeat("y", 60)}
	for _, h := range cases {
		t.Run(h, func(t *testing.T) {
			resp := env.do(http.MethodPost, "/api/admin/handles/block", "tok-admin",
				map[string]string{"handle": h, "reason": "test"})
			env.mustStatus(resp, http.StatusBadRequest)
			env.drain(resp)
		})
	}
}

// TestAdminBlockHandle_ShowsBlockedStatusInUserList confirms that the
// handle_blocked field is set correctly in the admin user listing.
func TestAdminBlockHandle_ShowsBlockedStatusInUserList(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")
	env.addUser("tok-charlie", "user-charlie", "charlie", "verified")

	// Verify charlie.
	resp := env.do(http.MethodPost, "/api/verify/start", "tok-charlie", map[string]string{"handle": "CharlieRSI"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)
	env.scraper.setProfile(&rsi.Profile{Bio: sr.Token})
	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-charlie", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	// Independently block their handle.
	resp = env.do(http.MethodPost, "/api/admin/handles/block", "tok-admin",
		map[string]string{"handle": "CharlieRSI", "reason": "test"})
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// User list should reflect the block.
	resp = env.do(http.MethodGet, "/api/admin/users", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var users []adminUserEntry
	env.decodeJSON(resp, &users)
	if len(users) == 0 {
		t.Fatal("expected at least 1 user")
	}
	if !users[0].HandleBlocked {
		t.Error("expected handle_blocked=true after blocking the handle")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Org logo management
// ────────────────────────────────────────────────────────────────────────────

// TestAdminListOrgs_Empty returns an empty list when the cache is empty.
func TestAdminListOrgs_Empty(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	resp := env.do(http.MethodGet, "/api/admin/orgs", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var orgs []adminOrgEntry
	env.decodeJSON(resp, &orgs)
	if len(orgs) != 0 {
		t.Fatalf("expected 0 orgs, got %d", len(orgs))
	}
}

// TestAdminBlockOrgLogo blocks and then unblocks an org logo via direct store seeding.
func TestAdminBlockOrgLogo(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	// Seed the org cache directly via the store.
	ctx := t.Context()
	err := env.st.UpsertOrgCache(ctx, &store.OrgCacheEntry{
		SID:       "SPAWO",
		Name:      "Test Org",
		LogoPath:  "",
		FetchedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed org cache: %v", err)
	}

	// Block the logo.
	resp := env.do(http.MethodPost, "/api/admin/orgs/SPAWO/block-logo", "tok-admin", nil)
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// Listing should show logo_blocked=true.
	resp = env.do(http.MethodGet, "/api/admin/orgs", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var orgs []adminOrgEntry
	env.decodeJSON(resp, &orgs)
	if len(orgs) != 1 {
		t.Fatalf("expected 1 org, got %d", len(orgs))
	}
	if !orgs[0].LogoBlocked {
		t.Error("expected logo_blocked=true after blocking")
	}

	// Unblock.
	resp = env.do(http.MethodDelete, "/api/admin/orgs/SPAWO/block-logo", "tok-admin", nil)
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// logo_blocked should now be false.
	resp = env.do(http.MethodGet, "/api/admin/orgs", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var orgsAfter []adminOrgEntry
	env.decodeJSON(resp, &orgsAfter)
	if orgsAfter[0].LogoBlocked {
		t.Error("expected logo_blocked=false after unblocking")
	}
}

// TestAdminBlockOrgLogo_UnknownSID returns 404 when the SID is not in cache.
func TestAdminBlockOrgLogo_UnknownSID(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	resp := env.do(http.MethodPost, "/api/admin/orgs/UNKNOWN/block-logo", "tok-admin", nil)
	env.mustStatus(resp, http.StatusNotFound)
	env.drain(resp)
}

// TestAdminBlockOrgLogo_InvalidSID returns 400 for SIDs that fail validation.
func TestAdminBlockOrgLogo_InvalidSID(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	// SID exceeding 16 chars: isValidSID returns false → 400 Bad Request.
	resp := env.do(http.MethodPost, "/api/admin/orgs/BADSIDTOOLONGVALUE/block-logo", "tok-admin", nil)
	env.mustStatus(resp, http.StatusBadRequest)
	env.drain(resp)
}

// ────────────────────────────────────────────────────────────────────────────
// Report review queue
// ────────────────────────────────────────────────────────────────────────────

// seedReport creates a report directly in the store.
func seedReport(t *testing.T, env *testEnv, reportType, target, reason string) string {
	t.Helper()
	id, err := newID()
	if err != nil {
		t.Fatalf("newID: %v", err)
	}
	report := &store.Report{
		ID:         id,
		Type:       reportType,
		Target:     target,
		Reason:     reason,
		ReporterIP: "127.0.0.1",
		CreatedAt:  time.Now().UTC(),
		Status:     "pending",
	}
	if err := env.st.CreateReport(t.Context(), report); err != nil {
		t.Fatalf("seed report: %v", err)
	}
	return id
}

// TestAdminListReports_Empty returns an empty list when there are no reports.
func TestAdminListReports_Empty(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	resp := env.do(http.MethodGet, "/api/admin/reports", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var reports []adminReportEntry
	env.decodeJSON(resp, &reports)
	if len(reports) != 0 {
		t.Fatalf("expected 0 reports, got %d", len(reports))
	}
}

// TestAdminListReports_FilterByStatus confirms that the status query param filters results.
func TestAdminListReports_FilterByStatus(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	id1 := seedReport(t, env, "user", "SomeHandle", "test reason here please")
	_ = seedReport(t, env, "org", "SPAWO", "another test reason here")

	// Mark the first report reviewed via the API.
	resp := env.do(http.MethodPost, "/api/admin/reports/"+id1+"/review", "tok-admin", nil)
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// Filter pending — should only return the second.
	resp = env.do(http.MethodGet, "/api/admin/reports?status=pending", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var pending []adminReportEntry
	env.decodeJSON(resp, &pending)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending report, got %d", len(pending))
	}
	if pending[0].Target != "SPAWO" {
		t.Errorf("expected target SPAWO, got %q", pending[0].Target)
	}

	// Filter reviewed — should only return the first.
	resp = env.do(http.MethodGet, "/api/admin/reports?status=reviewed", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var reviewed []adminReportEntry
	env.decodeJSON(resp, &reviewed)
	if len(reviewed) != 1 {
		t.Fatalf("expected 1 reviewed report, got %d", len(reviewed))
	}
	if reviewed[0].ID != id1 {
		t.Errorf("expected report %q to be reviewed", id1)
	}
}

// TestAdminReviewReport marks a report as reviewed.
func TestAdminReviewReport(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	id := seedReport(t, env, "user", "HandleToReview", "this is a test reason")

	resp := env.do(http.MethodPost, "/api/admin/reports/"+id+"/review", "tok-admin", nil)
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// Confirm status via listing.
	resp = env.do(http.MethodGet, "/api/admin/reports?status=reviewed", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var reports []adminReportEntry
	env.decodeJSON(resp, &reports)
	if len(reports) != 1 || reports[0].ID != id {
		t.Errorf("expected report %q to appear in reviewed list", id)
	}
	if reports[0].Status != "reviewed" {
		t.Errorf("expected status=reviewed, got %q", reports[0].Status)
	}
}

// TestAdminDismissReport marks a report as dismissed.
func TestAdminDismissReport(t *testing.T) {
	env := newTestEnv(t, false)
	addAdmin(env, "tok-admin", "user-admin", "admin")

	id := seedReport(t, env, "org", "LUGORG", "this org logo is inappropriate")

	resp := env.do(http.MethodPost, "/api/admin/reports/"+id+"/dismiss", "tok-admin", nil)
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	resp = env.do(http.MethodGet, "/api/admin/reports?status=dismissed", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var reports []adminReportEntry
	env.decodeJSON(resp, &reports)
	if len(reports) != 1 || reports[0].ID != id {
		t.Errorf("expected report %q in dismissed list", id)
	}
	if reports[0].Status != "dismissed" {
		t.Errorf("expected status=dismissed, got %q", reports[0].Status)
	}
}
