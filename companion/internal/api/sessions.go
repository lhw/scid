package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "__Host-scid_session"

type sessionEntry struct {
	AccessToken string
	ExpiresAt   time.Time
}

type sessionManager struct {
	secret   []byte
	mu       sync.Mutex
	sessions map[string]sessionEntry
	ops      uint64
}

func newSessionManager(secret string) *sessionManager {
	return &sessionManager{
		secret:   []byte(secret),
		sessions: make(map[string]sessionEntry),
	}
}

func (m *sessionManager) create(accessToken string, ttl time.Duration) (string, time.Time, error) {
	sessionID, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(ttl)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sessionID] = sessionEntry{AccessToken: accessToken, ExpiresAt: expiresAt}
	return m.sign(sessionID), expiresAt, nil
}

func (m *sessionManager) lookup(signedValue string) (string, bool) {
	sessionID, ok := m.verify(signedValue)
	if !ok {
		return "", false
	}

	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ops++
	m.cleanupLocked(now)

	entry, exists := m.sessions[sessionID]
	if !exists || !entry.ExpiresAt.After(now) {
		delete(m.sessions, sessionID)
		return "", false
	}
	return entry.AccessToken, true
}

func (m *sessionManager) delete(signedValue string) {
	sessionID, ok := m.verify(signedValue)
	if !ok {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

func (m *sessionManager) cleanupLocked(now time.Time) {
	if m.ops%256 != 0 {
		return
	}
	for sessionID, entry := range m.sessions {
		if !entry.ExpiresAt.After(now) {
			delete(m.sessions, sessionID)
		}
	}
}

func (m *sessionManager) sign(sessionID string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(sessionID))
	return sessionID + "." + hex.EncodeToString(mac.Sum(nil))
}

func (m *sessionManager) verify(signedValue string) (string, bool) {
	parts := strings.SplitN(signedValue, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	expected := m.sign(parts[0])
	if !hmac.Equal([]byte(expected), []byte(signedValue)) {
		return "", false
	}
	return parts[0], true
}

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setSessionCookie(w http.ResponseWriter, value string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   maxAge,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
