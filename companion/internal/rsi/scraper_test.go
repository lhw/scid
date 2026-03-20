package rsi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context"

	"github.com/lhw/scid/companion/internal/rsi"
)

// mockProfileHTML represents a realistic RSI public profile page.
// It mirrors the div structure found on robertsspaceindustries.com/en/citizens/<handle>.
const mockProfileHTML = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>CyFreeze - Roberts Space Industries</title></head>
<body>
  <div class="profile-wrapper">
    <div class="profile-header">
      <div class="profile-main-info">
        <div class="handle" data-bind="username">CyFreeze</div>
      </div>
    </div>

    <div class="profile-content">
      <div class="left-col">
        <div class="citizen-stats">

          <div class="entry">
            <div class="label">UEE Citizen Record</div>
            <div class="value">#40746</div>
          </div>

          <div class="entry">
            <div class="label">Enlisted</div>
            <div class="value">Oct 18, 2012</div>
          </div>

          <div class="entry">
            <div class="label">Location</div>
            <div class="value">Germany</div>
          </div>

        </div>
      </div>

      <div class="right-col">
        <div class="profile-about">
          <div class="bio">
            scid:a3f8c2d1e9b04765 This is my Star Citizen profile. I fly with SPAWO.
          </div>
        </div>
      </div>
    </div>
  </div>
</body>
</html>`

// mockProfileHTMLNoBio has a profile without the verification token.
const mockProfileHTMLNoBio = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>BadPilot - Roberts Space Industries</title></head>
<body>
  <div class="profile-content">
    <div class="citizen-stats">
      <div class="entry">
        <div class="label">UEE Citizen Record</div>
        <div class="value">#99999</div>
      </div>
      <div class="entry">
        <div class="label">Enlisted</div>
        <div class="value">Jan 01, 2020</div>
      </div>
    </div>
    <div class="bio">Just a regular citizen. No token here.</div>
  </div>
</body>
</html>`

// mockProfileHTMLAlternateLayout uses a different page structure to test
// fallback parsing paths (citizen-record class, data attributes, etc.).
const mockProfileHTMLAlternateLayout = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>AltUser - Roberts Space Industries</title></head>
<body>
  <div class="profile-content">
    <div class="citizen-record">
      <div class="number">#12345</div>
    </div>
    <div class="entry citizen-stat">
      <span class="label">Enlisted</span>
      <span class="value">Mar 15, 2015</span>
    </div>
    <div class="bio">
      scid:deadbeef12345678 AltUser bio text.
    </div>
  </div>
</body>
</html>`

// scraperFor returns a Scraper backed by a test HTTP server serving the given
// HTML for any path. The server is automatically closed at test cleanup.
func scraperFor(t *testing.T, statusCode int, body string) (*rsi.Scraper, string) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusCode)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(ts.Close)
	// Return scraper and the base URL (with trailing slash) pointing at the server.
	return rsi.NewWithBaseURL(ts.URL + "/"), ts.URL + "/"
}

func TestFetchProfile_Bio(t *testing.T) {
	s, _ := scraperFor(t, http.StatusOK, mockProfileHTML)
	profile, err := s.FetchProfile(context.Background(), "CyFreeze")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if !strings.Contains(profile.Bio, "scid:a3f8c2d1e9b04765") {
		t.Errorf("expected bio to contain token, got: %q", profile.Bio)
	}
}

func TestFetchProfile_CitizenRecord(t *testing.T) {
	s, _ := scraperFor(t, http.StatusOK, mockProfileHTML)
	profile, err := s.FetchProfile(context.Background(), "CyFreeze")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if profile.CitizenRecord != "#40746" {
		t.Errorf("expected CitizenRecord=#40746, got: %q", profile.CitizenRecord)
	}
}

func TestFetchProfile_Enlisted(t *testing.T) {
	s, _ := scraperFor(t, http.StatusOK, mockProfileHTML)
	profile, err := s.FetchProfile(context.Background(), "CyFreeze")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if profile.Enlisted != "Oct 18, 2012" {
		t.Errorf("expected Enlisted='Oct 18, 2012', got: %q", profile.Enlisted)
	}
}

func TestContainsToken_Found(t *testing.T) {
	s, _ := scraperFor(t, http.StatusOK, mockProfileHTML)
	profile, err := s.FetchProfile(context.Background(), "CyFreeze")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if !rsi.ContainsToken(profile, "scid:a3f8c2d1e9b04765") {
		t.Error("expected ContainsToken to return true")
	}
}

func TestContainsToken_WrongToken(t *testing.T) {
	s, _ := scraperFor(t, http.StatusOK, mockProfileHTML)
	profile, err := s.FetchProfile(context.Background(), "CyFreeze")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if rsi.ContainsToken(profile, "scid:ffffffffffffffff") {
		t.Error("expected ContainsToken to return false for wrong token")
	}
}

func TestFetchProfile_NotFound(t *testing.T) {
	s, _ := scraperFor(t, http.StatusNotFound, "Not Found")
	_, err := s.FetchProfile(context.Background(), "NoSuchPilot")
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestFetchProfile_AlternateLayout(t *testing.T) {
	s, _ := scraperFor(t, http.StatusOK, mockProfileHTMLAlternateLayout)
	profile, err := s.FetchProfile(context.Background(), "AltUser")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if !strings.Contains(profile.Bio, "scid:deadbeef12345678") {
		t.Errorf("expected bio to contain token, got: %q", profile.Bio)
	}
	if profile.Enlisted != "Mar 15, 2015" {
		t.Errorf("expected Enlisted='Mar 15, 2015', got: %q", profile.Enlisted)
	}
}

func TestFetchProfile_NoBio_CitizenRecord(t *testing.T) {
	s, _ := scraperFor(t, http.StatusOK, mockProfileHTMLNoBio)
	profile, err := s.FetchProfile(context.Background(), "BadPilot")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if profile.CitizenRecord != "#99999" {
		t.Errorf("expected CitizenRecord=#99999, got: %q", profile.CitizenRecord)
	}
	if profile.Enlisted != "Jan 01, 2020" {
		t.Errorf("expected Enlisted='Jan 01, 2020', got: %q", profile.Enlisted)
	}
}

func TestContainsToken_NoBio(t *testing.T) {
	s, _ := scraperFor(t, http.StatusOK, mockProfileHTMLNoBio)
	profile, err := s.FetchProfile(context.Background(), "BadPilot")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if rsi.ContainsToken(profile, "scid:a3f8c2d1e9b04765") {
		t.Error("expected ContainsToken to return false")
	}
}
