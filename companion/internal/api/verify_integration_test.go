package api

import (
	"net/http"
	"testing"

	"github.com/lhw/scid/companion/internal/pocketid"
	"github.com/lhw/scid/companion/internal/rsi"
)

// TestVerifyStart_RequiresAuth verifies that the endpoint rejects unauthenticated requests.
func TestVerifyStart_RequiresAuth(t *testing.T) {
	env := newTestEnv(t, false)
	resp := env.do(http.MethodPost, "/api/verify/start", "", map[string]string{"handle": "TestUser"})
	env.mustStatus(resp, http.StatusUnauthorized)
	env.drain(resp)
}

// TestVerifyStart_InvalidHandle rejects handles that don't match the format rule.
func TestVerifyStart_InvalidHandle(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice")

	tests := []struct {
		handle string
	}{
		{"ab"}, // too short
		{"this-handle-is-way-too-long-exceeding-sixty-characters-definitely-yes"},
		{"bad handle!"}, // spaces / special chars
		{""},
	}
	for _, tt := range tests {
		t.Run(tt.handle, func(t *testing.T) {
			resp := env.do(http.MethodPost, "/api/verify/start", "tok-alice", map[string]string{"handle": tt.handle})
			env.mustStatus(resp, http.StatusBadRequest)
			env.drain(resp)
		})
	}
}

// TestVerifyStart_Success returns a token for a valid handle.
func TestVerifyStart_Success(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice")

	resp := env.do(http.MethodPost, "/api/verify/start", "tok-alice", map[string]string{"handle": "AliceRSI"})
	env.mustStatus(resp, http.StatusOK)

	var sr startResponse
	env.decodeJSON(resp, &sr)

	if sr.Token == "" {
		t.Fatal("expected a non-empty verification token")
	}
	if !contains(sr.Token, "scid:") {
		t.Errorf("token should have scid: prefix, got %q", sr.Token)
	}
	if sr.ExpiresAt.IsZero() {
		t.Error("expected a non-zero expiry time")
	}
}

// TestVerifyStart_HandleConflict returns 409 when another user already owns the handle.
func TestVerifyStart_HandleConflict(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice")
	env.addUser("tok-bob", "user-bob", "bob")

	// Alice claims 'BattlePilot' first.
	resp := env.do(http.MethodPost, "/api/verify/start", "tok-alice", map[string]string{"handle": "BattlePilot"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)

	// Set mock scraper to put Alice's token in the bio.
	env.scraper.setProfile(&rsi.Profile{Bio: sr.Token})

	// Confirm Alice's verification so the handle is locked.
	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	// Bob tries to claim the same handle.
	resp = env.do(http.MethodPost, "/api/verify/start", "tok-bob", map[string]string{"handle": "BattlePilot"})
	env.mustStatus(resp, http.StatusConflict)
	env.drain(resp)
}

// TestVerifyConfirm_RequiresAuth blocks unauthenticated calls.
func TestVerifyConfirm_RequiresAuth(t *testing.T) {
	env := newTestEnv(t, false)
	resp := env.do(http.MethodPost, "/api/verify/confirm", "", nil)
	env.mustStatus(resp, http.StatusUnauthorized)
	env.drain(resp)
}

// TestVerifyConfirm_NoToken returns 400 when no pending verification exists.
func TestVerifyConfirm_NoToken(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice")

	resp := env.do(http.MethodPost, "/api/verify/confirm", "tok-alice", nil)
	env.mustStatus(resp, http.StatusBadRequest)
	env.drain(resp)
}

// TestVerifyConfirm_TokenNotInBio returns verified:false when the token is absent from the bio.
func TestVerifyConfirm_TokenNotInBio(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice")

	// Start verification.
	resp := env.do(http.MethodPost, "/api/verify/start", "tok-alice", map[string]string{"handle": "AliceRSI"})
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	// Mock profile has no token in the bio.
	env.scraper.setProfile(&rsi.Profile{Bio: "Hello, I fly spaceships."})

	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)

	var cr confirmResponse
	env.decodeJSON(resp, &cr)
	if cr.Verified {
		t.Error("expected verified=false when token is absent from bio")
	}
}

// TestVerifyConfirm_Success completes the full happy-path verification flow.
func TestVerifyConfirm_Success(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice")

	// Step 1 — start verification and capture the token.
	resp := env.do(http.MethodPost, "/api/verify/start", "tok-alice", map[string]string{"handle": "AliceRSI"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)

	// Step 2 — configure mock RSI to return a bio containing the token.
	env.scraper.setProfile(&rsi.Profile{
		Bio:           "My SCID token: " + sr.Token,
		CitizenRecord: "#99999",
		Enlisted:      "Jan 1, 2020",
	})

	// Step 3 — confirm verification.
	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)

	var cr confirmResponse
	env.decodeJSON(resp, &cr)

	if !cr.Verified {
		t.Fatal("expected verified=true")
	}
	if cr.Handle != "AliceRSI" {
		t.Errorf("expected handle=AliceRSI, got %q", cr.Handle)
	}

	// The "verified" group should now appear in mock PID for this user.
	groups := env.pid.getUserGroups("user-alice")
	found := false
	for _, g := range groups {
		if g.Name == "verified" {
			found = true
			break
		}
	}
	if !found {
		t.Error("user was not added to the 'verified' group after confirm")
	}
}

func TestVerifyConfirm_AddsStaffGroup(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-staff", "user-staff", "staff")

	resp := env.do(http.MethodPost, "/api/verify/start", "tok-staff", map[string]string{"handle": "Proxus-CIG"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)

	env.scraper.setProfile(&rsi.Profile{
		Bio:               "My SCID token: " + sr.Token,
		HasDeveloperBadge: true,
		CitizenRecord:     "#1406355",
		Enlisted:          "Jul 5, 2016",
	})

	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-staff", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	groups := env.pid.getUserGroups("user-staff")
	foundVerified := false
	foundStaff := false
	for _, g := range groups {
		switch g.Name {
		case "verified":
			foundVerified = true
		case "rsi:staff":
			foundStaff = true
		}
	}
	if !foundVerified {
		t.Error("expected verified group to be assigned")
	}
	if !foundStaff {
		t.Error("expected rsi:staff group to be assigned")
	}
}

func TestVerifyConfirm_DoesNotAddStaffGroupWithoutBadge(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-staff", "user-staff", "staff")

	resp := env.do(http.MethodPost, "/api/verify/start", "tok-staff", map[string]string{"handle": "Proxus-CIG"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)

	env.scraper.setProfile(&rsi.Profile{
		Bio:      "My SCID token: " + sr.Token,
		Enlisted: "Jul 5, 2016",
	})

	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-staff", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	groups := env.pid.getUserGroups("user-staff")
	for _, g := range groups {
		if g.Name == "rsi:staff" {
			t.Fatal("did not expect rsi:staff group without developer badge")
		}
	}
}

// TestVerifyStatus_Unauthenticated returns authenticated:false for no-auth requests.
func TestVerifyStatus_Unauthenticated(t *testing.T) {
	env := newTestEnv(t, false)

	resp := env.do(http.MethodGet, "/api/verify/status", "", nil)
	env.mustStatus(resp, http.StatusOK)

	var sr statusResponse
	env.decodeJSON(resp, &sr)
	if sr.Authenticated {
		t.Error("expected authenticated=false for unauthenticated request")
	}
}

// TestVerifyStatus_Authenticated returns user info for logged-in but unverified user.
func TestVerifyStatus_Authenticated(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-charlie", "user-charlie", "charlie")

	resp := env.do(http.MethodGet, "/api/verify/status", "tok-charlie", nil)
	env.mustStatus(resp, http.StatusOK)

	var sr statusResponse
	env.decodeJSON(resp, &sr)
	if !sr.Authenticated {
		t.Error("expected authenticated=true")
	}
	if sr.Verified {
		t.Error("expected verified=false for unverified user")
	}
	if sr.Username != "charlie" {
		t.Errorf("expected username=charlie, got %q", sr.Username)
	}
}

// TestVerifyStatus_PendingToken returns pending_handle when a start has been called.
func TestVerifyStatus_PendingToken(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice")

	// Start (but don't confirm) a verification.
	resp := env.do(http.MethodPost, "/api/verify/start", "tok-alice", map[string]string{"handle": "AliceHandle"})
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	resp = env.do(http.MethodGet, "/api/verify/status", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)

	var sr statusResponse
	env.decodeJSON(resp, &sr)
	if sr.PendingHandle != "AliceHandle" {
		t.Errorf("expected pending_handle=AliceHandle, got %q", sr.PendingHandle)
	}
	if sr.PendingExpiresAt == nil {
		t.Error("expected pending_expires_at to be set")
	}
}

// TestVerifyStatus_Verified returns verified:true and claim data for a verified user.
func TestVerifyStatus_Verified(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	// Pre-populate custom claims in mock PID as if the user was already verified.
	env.pid.mu.Lock()
	env.pid.users["user-alice"].claims = []pidClaim{
		{Key: "rsi_handle", Value: "AliceRSI"},
		{Key: "rsi_verified_at", Value: "2024-01-01T00:00:00Z"},
		{Key: "rsi_enlisted", Value: "2020-01-01"},
		{Key: "rsi_citizen_record", Value: "99999"},
	}
	env.pid.mu.Unlock()

	resp := env.do(http.MethodGet, "/api/verify/status", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)

	var sr statusResponse
	env.decodeJSON(resp, &sr)
	if !sr.Verified {
		t.Error("expected verified=true")
	}
	if sr.Handle != "AliceRSI" {
		t.Errorf("expected handle=AliceRSI, got %q", sr.Handle)
	}
	if sr.CitizenRecord != "99999" {
		t.Errorf("expected citizen_record=99999, got %q", sr.CitizenRecord)
	}
}

// TestVerifyFullFlow tests start → confirm → status in sequence.
func TestVerifyFullFlow(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-dave", "user-dave", "dave")

	// 1. Start
	resp := env.do(http.MethodPost, "/api/verify/start", "tok-dave", map[string]string{"handle": "DaveRSI"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)

	// 2. Place token in bio
	env.scraper.setProfile(&rsi.Profile{
		Bio:      sr.Token + " - my scid",
		Enlisted: "Oct 18, 2012",
	})

	// 3. Confirm
	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-dave", nil)
	env.mustStatus(resp, http.StatusOK)
	var cr confirmResponse
	env.decodeJSON(resp, &cr)
	if !cr.Verified {
		t.Fatalf("confirm returned verified=false: %+v", cr)
	}

	// 4. Status should reflect verified state (via group check)
	// Note: the status endpoint reads groups from mock PID, which now includes "verified"
	resp = env.do(http.MethodGet, "/api/verify/status", "tok-dave", nil)
	env.mustStatus(resp, http.StatusOK)
	var sr2 statusResponse
	env.decodeJSON(resp, &sr2)
	if !sr2.Verified {
		t.Error("status should show verified=true after confirm")
	}
}

// TestVerifyRefresh re-fetches profile data for a verified user.
func TestVerifyRefresh(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-eve", "user-eve", "eve", "verified")

	// Pre-populate claims so handleVerifyRefresh can find the handle.
	env.pid.mu.Lock()
	env.pid.users["user-eve"].claims = []pidClaim{
		{Key: "rsi_handle", Value: "EveRSI"},
		{Key: "rsi_verified_at", Value: "2024-01-01T00:00:00Z"},
	}
	env.pid.mu.Unlock()

	// Configure mock RSI with new profile data.
	env.scraper.setProfile(&rsi.Profile{
		Bio:           "Updated bio",
		Enlisted:      "Mar 5, 2021",
		CitizenRecord: "#12345",
	})

	resp := env.do(http.MethodPost, "/api/verify/refresh", "tok-eve", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)
}

// TestDeleteAccount removes the authenticated user from Pocket ID.
func TestDeleteAccount(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-frank", "user-frank", "frank")

	resp := env.do(http.MethodPost, "/api/account/delete", "tok-frank", nil)
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// Subsequent requests with the same token should fail (user deleted from mock PID).
	resp = env.do(http.MethodGet, "/api/verify/status", "tok-frank", nil)
	env.mustStatus(resp, http.StatusOK)
	var sr statusResponse
	env.decodeJSON(resp, &sr)
	if sr.Authenticated {
		t.Error("expected authenticated=false after account deletion")
	}
}

// contains returns true if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

// pidClaim is an alias used in tests to set user claims directly on the mock.
// It mirrors pocketid.CustomClaim without importing the package at test setup time.
type pidClaim = pocketid.CustomClaim

func TestVerifyConfirm_PreservesUnmanagedCustomClaims(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice")

	env.pid.mu.Lock()
	env.pid.users["user-alice"].claims = []pidClaim{{Key: "favorite_ship", Value: "Carrack"}}
	env.pid.mu.Unlock()

	resp := env.do(http.MethodPost, "/api/verify/start", "tok-alice", map[string]string{"handle": "AliceRSI"})
	env.mustStatus(resp, http.StatusOK)
	var sr startResponse
	env.decodeJSON(resp, &sr)

	env.scraper.setProfile(&rsi.Profile{
		Bio:           "Token: " + sr.Token,
		CitizenRecord: "#99999",
		Enlisted:      "Jan 1, 2020",
	})

	resp = env.do(http.MethodPost, "/api/verify/confirm", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	env.pid.mu.Lock()
	defer env.pid.mu.Unlock()
	claims := env.pid.users["user-alice"].claims
	for _, claim := range claims {
		if claim.Key == "favorite_ship" && claim.Value == "Carrack" {
			return
		}
	}
	t.Fatal("unmanaged custom claim was lost during verification")
}

func TestVerifyRefresh_PreservesUnmanagedCustomClaims(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-eve", "user-eve", "eve", "verified")

	env.pid.mu.Lock()
	env.pid.users["user-eve"].claims = []pidClaim{
		{Key: "rsi_handle", Value: "EveRSI"},
		{Key: "rsi_verified_at", Value: "2024-01-01T00:00:00Z"},
		{Key: "favorite_ship", Value: "Polaris"},
	}
	env.pid.mu.Unlock()

	env.scraper.setProfile(&rsi.Profile{
		Bio:           "Updated bio",
		Enlisted:      "Mar 5, 2021",
		CitizenRecord: "#12345",
	})

	resp := env.do(http.MethodPost, "/api/verify/refresh", "tok-eve", nil)
	env.mustStatus(resp, http.StatusOK)
	env.drain(resp)

	env.pid.mu.Lock()
	defer env.pid.mu.Unlock()
	claims := env.pid.users["user-eve"].claims
	for _, claim := range claims {
		if claim.Key == "favorite_ship" && claim.Value == "Polaris" {
			return
		}
	}
	t.Fatal("unmanaged custom claim was lost during refresh")
}

// TestVerifyStart_BlockedHandle verifies that a blocked RSI handle cannot be
// used to start the verification flow.
func TestVerifyStart_BlockedHandle(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-dave", "user-dave", "dave")

	// Block the handle directly via the store before attempting verification.
	if err := env.st.BlockHandle(t.Context(), "BlockedRSIHandle", "admin", "spam"); err != nil {
		t.Fatalf("block handle: %v", err)
	}

	resp := env.do(http.MethodPost, "/api/verify/start", "tok-dave",
map[string]string{"handle": "BlockedRSIHandle"})
	env.mustStatus(resp, http.StatusForbidden)
	env.drain(resp)
}
