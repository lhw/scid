package api

import (
	"fmt"
	"net/http"
	"testing"
)

// minAppRequest returns a minimal valid createAppRequest body.
func minAppRequest(name string) map[string]any {
	return map[string]any{
		"name":          name,
		"redirect_uris": []string{"https://example.com/callback"},
		"is_public":     true,
	}
}

// TestCreateApp_RequiresAuth blocks unauthenticated requests.
func TestCreateApp_RequiresAuth(t *testing.T) {
	env := newTestEnv(t, false)
	resp := env.do(http.MethodPost, "/api/apps", "", minAppRequest("My App"))
	env.mustStatus(resp, http.StatusUnauthorized)
	env.drain(resp)
}

// TestCreateApp_UnverifiedUserForbidden blocks users not in "verified" group.
func TestCreateApp_UnverifiedUserForbidden(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-unverified", "user-unverified", "unverified") // no groups

	resp := env.do(http.MethodPost, "/api/apps", "tok-unverified", minAppRequest("My App"))
	env.mustStatus(resp, http.StatusForbidden)
	env.drain(resp)
}

// TestCreateApp_AutoApproved: with RequireAppApproval=false the app is approved immediately.
func TestCreateApp_AutoApproved(t *testing.T) {
	env := newTestEnv(t, false /* no approval required */)
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Auto App"))
	env.mustStatus(resp, http.StatusCreated)

	var ar appResponse
	env.decodeJSON(resp, &ar)

	if ar.Status != "approved" {
		t.Errorf("expected status=approved, got %q", ar.Status)
	}
	if ar.ID == "" {
		t.Error("expected non-empty ID")
	}

	// Cleanup: delete the OIDC client we just created so the test is self-contained.
	t.Cleanup(func() {
		env.pid.mu.Lock()
		delete(env.pid.clients, ar.ID)
		env.pid.mu.Unlock()
	})
}

// TestCreateApp_PendingApproval: with RequireAppApproval=true the app is pending.
func TestCreateApp_PendingApproval(t *testing.T) {
	env := newTestEnv(t, true /* approval required */)
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Pending App"))
	env.mustStatus(resp, http.StatusCreated)

	var ar appResponse
	env.decodeJSON(resp, &ar)

	if ar.Status != "pending" {
		t.Errorf("expected status=pending, got %q", ar.Status)
	}
}

// TestCreateApp_PrivateClientSecret ensures the secret is returned for private clients.
func TestCreateApp_PrivateClientSecret(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	body := minAppRequest("Private App")
	body["is_public"] = false

	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", body)
	env.mustStatus(resp, http.StatusCreated)

	var ar appResponse
	env.decodeJSON(resp, &ar)

	if ar.ClientSecret == "" {
		t.Error("expected client_secret to be set for a private client")
	}
}

// TestListApps returns only the apps owned by the authenticated user.
func TestListApps(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice", "verified")
	env.addUser("tok-bob", "user-bob", "bob", "verified")

	// Alice creates two apps.
	for i := 0; i < 2; i++ {
		resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest(fmt.Sprintf("Alice App %d", i)))
		env.mustStatus(resp, http.StatusCreated)
		env.drain(resp)
	}

	// Bob creates one app.
	resp := env.do(http.MethodPost, "/api/apps", "tok-bob", minAppRequest("Bob App"))
	env.mustStatus(resp, http.StatusCreated)
	env.drain(resp)

	// Alice should see exactly 2 apps.
	resp = env.do(http.MethodGet, "/api/apps", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)
	var apps []appResponse
	env.decodeJSON(resp, &apps)
	if len(apps) != 2 {
		t.Errorf("Alice: expected 2 apps, got %d", len(apps))
	}

	// Bob should see exactly 1 app.
	resp = env.do(http.MethodGet, "/api/apps", "tok-bob", nil)
	env.mustStatus(resp, http.StatusOK)
	env.decodeJSON(resp, &apps)
	if len(apps) != 1 {
		t.Errorf("Bob: expected 1 app, got %d", len(apps))
	}
}

// TestDeleteApp_OwnerOnly prevents other users from deleting apps they don't own.
func TestDeleteApp_OwnerOnly(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-owner", "user-owner", "owner", "verified")
	env.addUser("tok-other", "user-other", "other", "verified")

	// Owner creates an app.
	resp := env.do(http.MethodPost, "/api/apps", "tok-owner", minAppRequest("Owner App"))
	env.mustStatus(resp, http.StatusCreated)
	var ar appResponse
	env.decodeJSON(resp, &ar)

	// Other user tries to delete it — should get 404 (hidden existence).
	resp = env.do(http.MethodDelete, "/api/apps/"+ar.ID, "tok-other", nil)
	env.mustStatus(resp, http.StatusNotFound)
	env.drain(resp)
}

// TestDeleteApp_Success allows the owner to delete their own app.
func TestDeleteApp_Success(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	// Create the app.
	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Deletable App"))
	env.mustStatus(resp, http.StatusCreated)
	var ar appResponse
	env.decodeJSON(resp, &ar)

	// Delete it.
	resp = env.do(http.MethodDelete, "/api/apps/"+ar.ID, "tok-alice", nil)
	env.mustStatus(resp, http.StatusNoContent)
	env.drain(resp)

	// It should no longer appear in the list.
	resp = env.do(http.MethodGet, "/api/apps", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)
	var apps []appResponse
	env.decodeJSON(resp, &apps)
	for _, a := range apps {
		if a.ID == ar.ID {
			t.Error("deleted app still appears in list")
		}
	}
}

// TestApproveApp_AdminOnly blocks non-admin users.
func TestApproveApp_AdminOnly(t *testing.T) {
	env := newTestEnv(t, true)
	env.addUser("tok-alice", "user-alice", "alice", "verified")
	env.addUser("tok-nonAdmin", "user-nonadmin", "nonadmin", "verified")

	// Alice creates a pending app.
	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Admin Test App"))
	env.mustStatus(resp, http.StatusCreated)
	var ar appResponse
	env.decodeJSON(resp, &ar)

	// Non-admin tries to approve.
	resp = env.do(http.MethodPost, "/api/admin/apps/"+ar.ID+"/approve", "tok-nonAdmin", nil)
	env.mustStatus(resp, http.StatusForbidden)
	env.drain(resp)
}

// TestApproveApp_Success allows an admin to approve a pending app.
func TestApproveApp_Success(t *testing.T) {
	env := newTestEnv(t, true)
	env.addUser("tok-alice", "user-alice", "alice", "verified")
	env.addUser("tok-admin", "user-admin", "admin", "admin")

	// Alice creates a pending app.
	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Approvable App"))
	env.mustStatus(resp, http.StatusCreated)
	var ar appResponse
	env.decodeJSON(resp, &ar)

	if ar.Status != "pending" {
		t.Fatalf("precondition: expected pending app, got %q", ar.Status)
	}

	// Admin approves it.
	resp = env.do(http.MethodPost, "/api/admin/apps/"+ar.ID+"/approve", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var approvedAR appResponse
	env.decodeJSON(resp, &approvedAR)

	if approvedAR.Status != "approved" {
		t.Errorf("expected status=approved after approval, got %q", approvedAR.Status)
	}
}

// TestApproveApp_AlreadyApproved returns 409 when trying to approve an already-approved app.
func TestApproveApp_AlreadyApproved(t *testing.T) {
	env := newTestEnv(t, false) // auto-approved
	env.addUser("tok-alice", "user-alice", "alice", "verified")
	env.addUser("tok-admin", "user-admin", "admin", "admin")

	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Already Approved"))
	env.mustStatus(resp, http.StatusCreated)
	var ar appResponse
	env.decodeJSON(resp, &ar)

	// Try to approve again.
	resp = env.do(http.MethodPost, "/api/admin/apps/"+ar.ID+"/approve", "tok-admin", nil)
	env.mustStatus(resp, http.StatusConflict)
	env.drain(resp)
}

// TestRejectApp_Success allows an admin to reject a pending app with a reason.
func TestRejectApp_Success(t *testing.T) {
	env := newTestEnv(t, true)
	env.addUser("tok-alice", "user-alice", "alice", "verified")
	env.addUser("tok-admin", "user-admin", "admin", "admin")

	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Rejectible App"))
	env.mustStatus(resp, http.StatusCreated)
	var ar appResponse
	env.decodeJSON(resp, &ar)

	resp = env.do(http.MethodPost, "/api/admin/apps/"+ar.ID+"/reject", "tok-admin",
		map[string]string{"reason": "does not meet guidelines"})
	env.mustStatus(resp, http.StatusOK)
	var rejectedAR appResponse
	env.decodeJSON(resp, &rejectedAR)

	if rejectedAR.Status != "rejected" {
		t.Errorf("expected status=rejected, got %q", rejectedAR.Status)
	}
	if rejectedAR.RejectionReason != "does not meet guidelines" {
		t.Errorf("expected rejection_reason to be set, got %q", rejectedAR.RejectionReason)
	}
}

// TestDirectoryListing returns only approved+listed apps.
func TestDirectoryListing(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	// Create a listed app (auto-approved + listed).
	listedBody := map[string]any{
		"name":          "Listed App",
		"redirect_uris": []string{"https://example.com/callback"},
		"launch_url":    "https://example.com/",
		"is_public":     true,
		"listed":        true,
	}
	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", listedBody)
	env.mustStatus(resp, http.StatusCreated)
	var listed appResponse
	env.decodeJSON(resp, &listed)

	// Create an unlisted app.
	resp = env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Unlisted App"))
	env.mustStatus(resp, http.StatusCreated)
	env.drain(resp)

	// The public directory should contain only the listed app.
	resp = env.do(http.MethodGet, "/api/apps/directory", "", nil)
	env.mustStatus(resp, http.StatusOK)
	var dir []appResponse
	env.decodeJSON(resp, &dir)

	found := false
	for _, a := range dir {
		if a.ID == listed.ID {
			found = true
		}
		// Unlisted apps should never appear.
		if a.Name == "Unlisted App" {
			t.Error("unlisted app appeared in directory")
		}
	}
	if !found {
		t.Error("listed app not found in directory")
	}
}

// TestMaxAppsPerUser enforces the 5-app limit per verified user.
func TestMaxAppsPerUser(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	// Create 5 apps (the maximum).
	for i := 0; i < maxAppsPerUser; i++ {
		resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest(fmt.Sprintf("App %d", i)))
		env.mustStatus(resp, http.StatusCreated)
		env.drain(resp)
	}

	// The 6th app should be rejected.
	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("App 6"))
	env.mustStatus(resp, http.StatusUnprocessableEntity)
	env.drain(resp)
}

// TestRotateSecret returns a new client_secret.
func TestRotateSecret(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	// Create a private app.
	body := minAppRequest("Secret App")
	body["is_public"] = false
	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", body)
	env.mustStatus(resp, http.StatusCreated)
	var ar appResponse
	env.decodeJSON(resp, &ar)

	// Rotate the secret.
	resp = env.do(http.MethodPost, "/api/apps/"+ar.ID+"/secret", "tok-alice", nil)
	env.mustStatus(resp, http.StatusOK)

	var result map[string]string
	env.decodeJSON(resp, &result)
	if result["client_secret"] == "" {
		t.Error("expected client_secret to be non-empty after rotation")
	}
}

// TestUpdateApp allows the owner to update app details.
func TestUpdateApp(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-alice", "user-alice", "alice", "verified")

	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Original Name"))
	env.mustStatus(resp, http.StatusCreated)
	var ar appResponse
	env.decodeJSON(resp, &ar)

	updated := map[string]any{
		"name":          "Updated Name",
		"redirect_uris": []string{"https://example.com/callback"},
		"is_public":     true,
	}
	resp = env.do(http.MethodPut, "/api/apps/"+ar.ID, "tok-alice", updated)
	env.mustStatus(resp, http.StatusOK)

	var updatedAR appResponse
	env.decodeJSON(resp, &updatedAR)
	if updatedAR.Name != "Updated Name" {
		t.Errorf("expected name=Updated Name, got %q", updatedAR.Name)
	}
}

// TestAdminListApps allows admins to see all apps.
func TestAdminListApps(t *testing.T) {
	env := newTestEnv(t, true)
	env.addUser("tok-alice", "user-alice", "alice", "verified")
	env.addUser("tok-admin", "user-admin", "admin", "admin")

	resp := env.do(http.MethodPost, "/api/apps", "tok-alice", minAppRequest("Admin Visible App"))
	env.mustStatus(resp, http.StatusCreated)
	var ar appResponse
	env.decodeJSON(resp, &ar)

	resp = env.do(http.MethodGet, "/api/admin/apps", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var apps []appResponse
	env.decodeJSON(resp, &apps)
	if len(apps) == 0 {
		t.Error("admin should see at least one app")
	}
	// Non-admin should be forbidden.
	resp = env.do(http.MethodGet, "/api/admin/apps", "tok-alice", nil)
	env.mustStatus(resp, http.StatusForbidden)
	env.drain(resp)
}
