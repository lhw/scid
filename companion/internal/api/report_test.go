package api

import (
	"net/http"
	"strings"
	"testing"
)

// TestSubmitReport_User_Success submits a valid user report and expects 201
// with the report ID in the response body.
func TestSubmitReport_User_Success(t *testing.T) {
	env := newTestEnv(t, false)

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "user",
		"target": "ReportedUser",
		"reason": "This user is harassing other players repeatedly.",
	})
	env.mustStatus(resp, http.StatusCreated)

	var body struct {
		ID string `json:"id"`
	}
	env.decodeJSON(resp, &body)
	if body.ID == "" {
		t.Error("expected a non-empty report ID in response")
	}
}

// TestSubmitReport_Org_Success submits a valid org report; the target SID
// should be upper-cased automatically.
func TestSubmitReport_Org_Success(t *testing.T) {
	env := newTestEnv(t, false)

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "org",
		"target": "spawo",
		"reason": "This organisation's logo violates community standards clearly.",
	})
	env.mustStatus(resp, http.StatusCreated)

	var body struct {
		ID string `json:"id"`
	}
	env.decodeJSON(resp, &body)
	if body.ID == "" {
		t.Error("expected a non-empty report ID in response")
	}
}

// TestSubmitReport_StoredInDB verifies that a submitted report can be retrieved
// from the store after submission.
func TestSubmitReport_StoredInDB(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-admin", "user-admin", "admin", "admin")

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "user",
		"target": "StoredUser",
		"reason": "Multiple confirmed reports of griefing behaviour.",
	})
	env.mustStatus(resp, http.StatusCreated)
	var body struct {
		ID string `json:"id"`
	}
	env.decodeJSON(resp, &body)

	// Fetch via admin listing to confirm storage.
	resp = env.do(http.MethodGet, "/api/admin/reports", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var reports []adminReportEntry
	env.decodeJSON(resp, &reports)

	found := false
	for _, r := range reports {
		if r.ID == body.ID && r.Target == "StoredUser" && r.Type == "user" {
			found = true
		}
	}
	if !found {
		t.Errorf("submitted report %q not found in admin listing", body.ID)
	}
}

// TestSubmitReport_InvalidType rejects unknown report types.
func TestSubmitReport_InvalidType(t *testing.T) {
	env := newTestEnv(t, false)

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "ban",
		"target": "SomeHandle",
		"reason": "This is a long enough reason string for the test.",
	})
	env.mustStatus(resp, http.StatusBadRequest)
	env.drain(resp)
}

// TestSubmitReport_MissingTarget rejects reports with an empty target.
func TestSubmitReport_MissingTarget(t *testing.T) {
	env := newTestEnv(t, false)

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "user",
		"target": "",
		"reason": "This is a long enough reason string for the test.",
	})
	env.mustStatus(resp, http.StatusBadRequest)
	env.drain(resp)
}

// TestSubmitReport_InvalidUserHandle rejects user reports with a malformed handle.
func TestSubmitReport_InvalidUserHandle(t *testing.T) {
	env := newTestEnv(t, false)

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "user",
		"target": "x!", // contains special character
		"reason": "This is a long enough reason string for the test.",
	})
	env.mustStatus(resp, http.StatusBadRequest)
	env.drain(resp)
}

// TestSubmitReport_InvalidOrgSID rejects org reports with a malformed SID.
func TestSubmitReport_InvalidOrgSID(t *testing.T) {
	env := newTestEnv(t, false)

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "org",
		"target": "bad-sid!", // contains invalid characters
		"reason": "This is a long enough reason string for the test.",
	})
	env.mustStatus(resp, http.StatusBadRequest)
	env.drain(resp)
}

// TestSubmitReport_ReasonTooShort rejects reasons shorter than 10 characters.
func TestSubmitReport_ReasonTooShort(t *testing.T) {
	env := newTestEnv(t, false)

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "user",
		"target": "SomeUser",
		"reason": "short",
	})
	env.mustStatus(resp, http.StatusBadRequest)
	env.drain(resp)
}

// TestSubmitReport_ReasonTooLong rejects reasons over 2000 characters.
func TestSubmitReport_ReasonTooLong(t *testing.T) {
	env := newTestEnv(t, false)

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "user",
		"target": "SomeUser",
		"reason": strings.Repeat("x", 2001),
	})
	env.mustStatus(resp, http.StatusBadRequest)
	env.drain(resp)
}

// TestSubmitReport_OrgTargetUppercased verifies the SID is stored in uppercase.
func TestSubmitReport_OrgTargetUppercased(t *testing.T) {
	env := newTestEnv(t, false)
	env.addUser("tok-admin", "user-admin", "admin", "admin")

	resp := env.do(http.MethodPost, "/api/report", "", map[string]string{
		"type":   "org",
		"target": "lwi", // lowercase — should be normalised to LWI
		"reason": "Org logo contains inappropriate content clearly.",
	})
	env.mustStatus(resp, http.StatusCreated)
	var body struct {
		ID string `json:"id"`
	}
	env.decodeJSON(resp, &body)

	// Confirm the stored target is uppercase.
	resp = env.do(http.MethodGet, "/api/admin/reports", "tok-admin", nil)
	env.mustStatus(resp, http.StatusOK)
	var reports []adminReportEntry
	env.decodeJSON(resp, &reports)

	for _, r := range reports {
		if r.ID == body.ID {
			if r.Target != "LWI" {
				t.Errorf("expected target LWI (uppercased), got %q", r.Target)
			}
			return
		}
	}
	t.Errorf("report %q not found in admin listing", body.ID)
}
