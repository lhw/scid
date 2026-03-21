package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/lhw/scid/companion/internal/config"
	"github.com/lhw/scid/companion/internal/pocketid"
	"github.com/lhw/scid/companion/internal/rsi"
	"github.com/lhw/scid/companion/internal/store"
)

// ----------------------------------------------------------------------------
// Mock RSI scraper
// ----------------------------------------------------------------------------

type mockRSIScraper struct {
	mu       sync.Mutex
	profile  *rsi.Profile
	orgs     []rsi.OrgInfo
	fetchErr error
}

func (m *mockRSIScraper) setProfile(p *rsi.Profile) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profile = p
}

func (m *mockRSIScraper) FetchProfile(_ context.Context, handle string) (*rsi.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	if m.profile == nil {
		return &rsi.Profile{Handle: handle, Bio: ""}, nil
	}
	p := *m.profile
	p.Handle = handle
	return &p, nil
}

func (m *mockRSIScraper) FetchOrgs(_ context.Context, _ string) ([]rsi.OrgInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.orgs, nil
}

// ----------------------------------------------------------------------------
// Mock Pocket ID server
// ----------------------------------------------------------------------------

type pidUser struct {
	id       string
	username string
	email    string
	groupIDs []string
	claims   []pocketid.CustomClaim
}

type pidGroup struct {
	id           string
	name         string
	friendlyName string
}

type pidOIDCClient struct {
	pocketid.OIDCClient
	secret string
}

type mockPocketID struct {
	mu           sync.Mutex
	idSeq        int64
	tokens       map[string]string         // bearer token → userID
	users        map[string]*pidUser       // userID → user
	groupsByName map[string]*pidGroup      // name → group
	groupsByID   map[string]*pidGroup      // id → group
	clients      map[string]*pidOIDCClient // clientID → client
}

func newMockPocketID() *mockPocketID {
	return &mockPocketID{
		tokens:       make(map[string]string),
		users:        make(map[string]*pidUser),
		groupsByName: make(map[string]*pidGroup),
		groupsByID:   make(map[string]*pidGroup),
		clients:      make(map[string]*pidOIDCClient),
	}
}

func (m *mockPocketID) genID() string {
	n := atomic.AddInt64(&m.idSeq, 1)
	return fmt.Sprintf("id-%d", n)
}

// addUser registers a user reachable via the given bearer token.
func (m *mockPocketID) addUser(token, userID, username string, groupNames ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	groupIDs := make([]string, 0, len(groupNames))
	for _, name := range groupNames {
		g := m.ensureGroupLocked(name, name)
		groupIDs = append(groupIDs, g.id)
	}
	m.tokens[token] = userID
	m.users[userID] = &pidUser{
		id:       userID,
		username: username,
		email:    username + "@test.example",
		groupIDs: groupIDs,
	}
}

// ensureGroupLocked finds or creates a group by name. Caller must hold m.mu.
func (m *mockPocketID) ensureGroupLocked(name, friendlyName string) *pidGroup {
	if g, ok := m.groupsByName[name]; ok {
		return g
	}
	id := fmt.Sprintf("grp-%d", atomic.AddInt64(&m.idSeq, 1))
	g := &pidGroup{id: id, name: name, friendlyName: friendlyName}
	m.groupsByName[name] = g
	m.groupsByID[id] = g
	return g
}

func (m *mockPocketID) buildRouter() http.Handler {
	r := chi.NewRouter()

	// OIDC discovery — used by the OIDC client for ping, token exchange, and userinfo.
	r.Get("/.well-known/openid-configuration", func(w http.ResponseWriter, req *http.Request) {
		baseURL := "http://" + req.Host
		writeJSON(w, http.StatusOK, map[string]any{
			"issuer":                 baseURL,
			"authorization_endpoint": baseURL + "/authorize",
			"token_endpoint":         baseURL + "/api/oidc/token",
			"userinfo_endpoint":      baseURL + "/api/oidc/userinfo",
		})
	})

	// Token exchange — used by auth callback tests.
	r.Post("/api/oidc/token", func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		code := req.Form.Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		_, ok := m.tokens[code]
		m.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token": code,
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})

	// Userinfo — used to authenticate bearer tokens
	r.Get("/api/oidc/userinfo", func(w http.ResponseWriter, req *http.Request) {
		auth := req.Header.Get("Authorization")
		const prefix = "Bearer "
		if len(auth) <= len(prefix) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		token := auth[len(prefix):]
		m.mu.Lock()
		userID, ok := m.tokens[token]
		m.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		m.mu.Lock()
		u := m.users[userID]
		m.mu.Unlock()
		if u == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"sub":                u.id,
			"preferred_username": u.username,
			"email":              u.email,
		})
	})

	// GET /api/users/{id}/groups
	r.Get("/api/users/{id}/groups", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "id")
		writeJSON(w, http.StatusOK, m.getUserGroups(userID))
	})

	// PUT /api/users/{id}/user-groups
	r.Put("/api/users/{id}/user-groups", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "id")
		var body struct {
			UserGroupIDs []string `json:"userGroupIds"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		if u, ok := m.users[userID]; ok {
			u.groupIDs = body.UserGroupIDs
		}
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	// PUT /api/users/{id}/profile-picture — no-op
	r.Put("/api/users/{id}/profile-picture", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(io.Discard, req.Body) //nolint:errcheck
		w.WriteHeader(http.StatusNoContent)
	})

	// GET /api/users/{id}
	r.Get("/api/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "id")
		m.mu.Lock()
		u, ok := m.users[userID]
		m.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, pocketid.UserDetail{
			ID:           u.id,
			Username:     u.username,
			Email:        u.email,
			CustomClaims: u.claims,
		})
	})

	// DELETE /api/users/{id}
	r.Delete("/api/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "id")
		m.mu.Lock()
		delete(m.users, userID)
		for tok, uid := range m.tokens {
			if uid == userID {
				delete(m.tokens, tok)
			}
		}
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	// GET /api/user-groups?search=...
	r.Get("/api/user-groups", func(w http.ResponseWriter, req *http.Request) {
		search := req.URL.Query().Get("search")
		m.mu.Lock()
		defer m.mu.Unlock()
		var data []pocketid.Group
		for name, g := range m.groupsByName {
			if search == "" || name == search {
				data = append(data, pocketid.Group{ID: g.id, Name: g.name, FriendlyName: g.friendlyName})
			}
		}
		if data == nil {
			data = []pocketid.Group{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": data})
	})

	// POST /api/user-groups
	r.Post("/api/user-groups", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Name         string `json:"name"`
			FriendlyName string `json:"friendlyName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		g := m.ensureGroupLocked(body.Name, body.FriendlyName)
		m.mu.Unlock()
		writeJSON(w, http.StatusCreated, pocketid.Group{ID: g.id, Name: g.name, FriendlyName: g.friendlyName})
	})

	// PUT /api/custom-claims/user/{id}
	r.Put("/api/custom-claims/user/{id}", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "id")
		var claims []pocketid.CustomClaim
		if err := json.NewDecoder(req.Body).Decode(&claims); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		if u, ok := m.users[userID]; ok {
			u.claims = claims
		}
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	// POST /api/signup-tokens
	r.Post("/api/signup-tokens", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusCreated, pocketid.SignupToken{
			ID:    m.genID(),
			Token: "signup-" + m.genID(),
		})
	})

	// POST /api/oidc/clients
	r.Post("/api/oidc/clients", func(w http.ResponseWriter, req *http.Request) {
		var params pocketid.OIDCClientParams
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		id := m.genID()
		logoutURIs := params.LogoutCallbackURLs
		if logoutURIs == nil {
			logoutURIs = []string{}
		}
		callbackURLs := params.CallbackURLs
		if callbackURLs == nil {
			callbackURLs = []string{}
		}
		c := &pidOIDCClient{
			OIDCClient: pocketid.OIDCClient{
				ID:                 id,
				Name:               params.Name,
				LaunchURL:          params.LaunchURL,
				IsPublic:           params.IsPublic,
				PkceEnabled:        params.PkceEnabled,
				CallbackURLs:       callbackURLs,
				LogoutCallbackURLs: logoutURIs,
				AllowedUserGroups:  []pocketid.Group{},
			},
		}
		m.clients[id] = c
		m.mu.Unlock()
		writeJSON(w, http.StatusCreated, c.OIDCClient)
	})

	// POST /api/oidc/clients/{id}/secret
	r.Post("/api/oidc/clients/{id}/secret", func(w http.ResponseWriter, req *http.Request) {
		clientID := chi.URLParam(req, "id")
		m.mu.Lock()
		secret := ""
		if c, ok := m.clients[clientID]; ok {
			c.secret = "secret-" + clientID
			secret = c.secret
		}
		m.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"secret": secret})
	})

	// PUT /api/oidc/clients/{id}/allowed-user-groups
	r.Put("/api/oidc/clients/{id}/allowed-user-groups", func(w http.ResponseWriter, req *http.Request) {
		clientID := chi.URLParam(req, "id")
		var body struct {
			UserGroupIDs []string `json:"userGroupIds"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		c, ok := m.clients[clientID]
		if ok {
			groups := make([]pocketid.Group, 0, len(body.UserGroupIDs))
			for _, gid := range body.UserGroupIDs {
				if g, gok := m.groupsByID[gid]; gok {
					groups = append(groups, pocketid.Group{ID: g.id, Name: g.name, FriendlyName: g.friendlyName})
				}
			}
			c.AllowedUserGroups = groups
			c.IsGroupRestricted = len(groups) > 0
		}
		var resp pocketid.OIDCClient
		if ok {
			resp = c.OIDCClient
		}
		m.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
	})

	// POST /api/oidc/clients/{id}/logo — no-op
	r.Post("/api/oidc/clients/{id}/logo", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(io.Discard, req.Body) //nolint:errcheck
		w.WriteHeader(http.StatusNoContent)
	})

	// GET /api/oidc/clients/{id}
	r.Get("/api/oidc/clients/{id}", func(w http.ResponseWriter, req *http.Request) {
		clientID := chi.URLParam(req, "id")
		m.mu.Lock()
		c, ok := m.clients[clientID]
		m.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, c.OIDCClient)
	})

	// PUT /api/oidc/clients/{id}
	r.Put("/api/oidc/clients/{id}", func(w http.ResponseWriter, req *http.Request) {
		clientID := chi.URLParam(req, "id")
		var params pocketid.OIDCClientParams
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		c, ok := m.clients[clientID]
		if ok {
			callbackURLs := params.CallbackURLs
			if callbackURLs == nil {
				callbackURLs = []string{}
			}
			logoutURIs := params.LogoutCallbackURLs
			if logoutURIs == nil {
				logoutURIs = []string{}
			}
			c.Name = params.Name
			c.LaunchURL = params.LaunchURL
			c.IsPublic = params.IsPublic
			c.PkceEnabled = params.PkceEnabled
			c.CallbackURLs = callbackURLs
			c.LogoutCallbackURLs = logoutURIs
		}
		var resp pocketid.OIDCClient
		if ok {
			resp = c.OIDCClient
		}
		m.mu.Unlock()
		writeJSON(w, http.StatusOK, resp)
	})

	// DELETE /api/oidc/clients/{id}
	r.Delete("/api/oidc/clients/{id}", func(w http.ResponseWriter, req *http.Request) {
		clientID := chi.URLParam(req, "id")
		m.mu.Lock()
		delete(m.clients, clientID)
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	return r
}

func (m *mockPocketID) getUserGroups(userID string) []pocketid.Group {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return []pocketid.Group{}
	}
	groups := make([]pocketid.Group, 0, len(u.groupIDs))
	for _, gid := range u.groupIDs {
		if g, ok := m.groupsByID[gid]; ok {
			groups = append(groups, pocketid.Group{ID: g.id, Name: g.name, FriendlyName: g.friendlyName})
		}
	}
	return groups
}

// ----------------------------------------------------------------------------
// Test environment
// ----------------------------------------------------------------------------

type testEnv struct {
	t       *testing.T
	srv     *httptest.Server
	pid     *mockPocketID
	scraper *mockRSIScraper
	st      *store.Store
}

// newTestEnv creates a complete test environment with in-memory SQLite, mock
// Pocket ID server, mock RSI scraper, and a running companion HTTP server.
// Resources are automatically cleaned up via t.Cleanup.
func newTestEnv(t *testing.T, requireApproval bool) *testEnv {
	t.Helper()

	pidMock := newMockPocketID()
	pidSrv := httptest.NewServer(pidMock.buildRouter())
	t.Cleanup(pidSrv.Close)

	scraperMock := &mockRSIScraper{}

	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	cfg := &config.Config{
		PocketIDInternalURL: pidSrv.URL,
		PocketIDAdminAPIKey: "test-admin-key",
		OIDCIssuerURL:       pidSrv.URL,
		OIDCClientID:        "scid-frontend",
		SessionSecretKey:    "test-session-secret-key-32bytes!",
		SessionTTL:          24 * 60 * 60 * 1e9, // 24h
		SessionCookieSecure: false,
		RequireAppApproval:  requireApproval,
	}

	companion := New(cfg, st)
	companion.scraper = scraperMock

	companionSrv := httptest.NewServer(companion)
	t.Cleanup(companionSrv.Close)

	return &testEnv{
		t:       t,
		srv:     companionSrv,
		pid:     pidMock,
		scraper: scraperMock,
		st:      st,
	}
}

// addUser registers a user in the mock PocketID server.
// Subsequent requests with Authorization: Bearer <token> will be authenticated
// as this user.
func (e *testEnv) addUser(token, id, username string, groups ...string) {
	e.pid.addUser(token, id, username, groups...)
}

// do performs an HTTP request to the companion server.
// token may be "" for unauthenticated requests.
// body may be nil for requests without a body.
func (e *testEnv) do(method, path, token string, body any) *http.Response {
	e.t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, bodyReader)
	if err != nil {
		e.t.Fatalf("build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("perform request %s %s: %v", method, path, err)
	}
	return resp
}

// mustStatus fails the test if resp does not have the expected HTTP status.
// On failure it reads and logs the response body for debugging.
func (e *testEnv) mustStatus(resp *http.Response, want int) {
	e.t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		e.t.Fatalf("expected HTTP %d, got %d; body: %s", want, resp.StatusCode, body)
	}
}

// decodeJSON reads and closes resp.Body, decoding JSON into dest.
func (e *testEnv) decodeJSON(resp *http.Response, dest any) {
	e.t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		e.t.Fatalf("decode response: %v", err)
	}
}

// drain reads and discards the response body.
func (e *testEnv) drain(resp *http.Response) {
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()
}
