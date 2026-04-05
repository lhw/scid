package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/lhw/scid/companion/internal/config"
	"github.com/lhw/scid/companion/internal/frontend"
	"github.com/lhw/scid/companion/internal/mailer"
	"github.com/lhw/scid/companion/internal/oidcclient"
	"github.com/lhw/scid/companion/internal/pocketid"
	"github.com/lhw/scid/companion/internal/rsi"
	"github.com/lhw/scid/companion/internal/store"
)

// Server wires together dependencies and the HTTP router.
type Server struct {
	cfg      *config.Config
	store    *store.Store
	auth     *oidcclient.Client
	pid      *pocketid.Client
	scraper  rsi.RSIScraper
	limiter  *rateLimiter
	sessions *scs.SessionManager
	mailer   *mailer.Mailer
	router   *chi.Mux
}

// New creates a configured Server ready to serve HTTP.
func New(cfg *config.Config, st *store.Store) *Server {
	sessionManager := scs.New()
	if st.Driver() == "postgres" {
		sessionManager.Store = postgresstore.NewWithCleanupInterval(st.DB(), 0)
	} else {
		sessionManager.Store = sqlite3store.NewWithCleanupInterval(st.DB(), 0)
	}
	sessionManager.Lifetime = cfg.SessionTTL
	sessionManager.Cookie.Name = sessionCookieName(cfg.SessionCookieSecure)
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = cfg.SessionCookieSecure
	sessionManager.Cookie.Persist = true
	sessionManager.HashTokenInStore = true

	s := &Server{
		cfg:      cfg,
		store:    st,
		auth:     oidcclient.New(cfg.OIDCIssuerURL, cfg.OIDCClientID, cfg.OIDCClientSecret),
		pid:      pocketid.New(cfg.PocketIDInternalURL, cfg.PocketIDAdminAPIKey),
		scraper:  rsi.New(),
		limiter:  newRateLimiter(),
		sessions: sessionManager,
		mailer: mailer.New(mailer.Config{
			Host:       cfg.SMTPHost,
			Port:       cfg.SMTPPort,
			User:       cfg.SMTPUser,
			Password:   cfg.SMTPPassword,
			From:       cfg.SMTPFrom,
			AdminEmail: cfg.SMTPAdminEmail,
		}),
	}
	s.router = s.buildRouter()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(s.trustedRealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(s.sessions.LoadAndSave)
	r.Use(prometheusMiddleware)

	// CORS — origins are configured via CORS_ALLOWED_ORIGINS (see config).
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// /metrics is served on the companion's internal port only and is not
	// proxied through Caddy — safe for Prometheus scraping within the stack.
	r.Mount("/metrics", prometheusHandler())
	r.Get("/api/health", s.handleHealth)
	// Status is intentionally public — returns {verified:false} when unauthenticated
	// so the home page can load without a login.
	r.Get("/api/verify/status", s.handleVerifyStatus)
	// Signup token — public endpoint that creates a 1-use Pocket ID registration token.
	// The frontend uses the token to redirect new users to Pocket ID's /signup page.
	r.With(s.publicRateLimitMiddleware("signup-token", 5, time.Minute)).Post("/api/auth/signup-token", s.handleSignupToken)
	r.With(s.publicRateLimitMiddleware("auth-callback", 20, 10*time.Minute)).Post("/api/auth/callback", s.handleAuthCallback)
	r.Post("/api/auth/logout", s.handleAuthLogout)
	// Org logo — serves cached org logos by SID (public, browser-cached).
	r.Get("/api/orgs/{sid}/logo", s.handleOrgLogo)
	// OIDC client logo — proxy from Pocket ID so the browser loads it same-origin.
	r.Get("/api/oidc/clients/{id}/logo", s.handleOIDCClientLogo)
	// Public app directory — lists approved apps that have opted into the directory.
	r.Get("/api/apps/directory", s.handleListDirectoryApps)
	// Public report form — accepts abuse reports for users/orgs (Turnstile required in prod).
	r.With(s.publicRateLimitMiddleware("report", 5, 10*time.Minute)).Post("/api/report", s.handleSubmitReport)

	r.Group(func(r chi.Router) {
		r.Use(s.bearerAuthMiddleware)
		r.With(s.authenticatedRateLimitMiddleware("verify-start", 6, 10*time.Minute)).Post("/api/verify/start", s.handleVerifyStart)
		r.With(s.authenticatedRateLimitMiddleware("verify-confirm", 12, 10*time.Minute)).Post("/api/verify/confirm", s.handleVerifyConfirm)
		r.With(s.authenticatedRateLimitMiddleware("verify-refresh", 6, time.Hour)).Post("/api/verify/refresh", s.handleVerifyRefresh)
		r.Post("/api/account/delete", s.handleDeleteAccount)
		r.With(s.authenticatedRateLimitMiddleware("apps-create", 10, time.Hour)).Post("/api/apps", s.handleCreateApp)
		r.Get("/api/apps", s.handleListApps)
		r.Get("/api/apps/{id}", s.handleGetApp)
		r.Put("/api/apps/{id}", s.handleUpdateApp)
		r.Delete("/api/apps/{id}", s.handleDeleteApp)
		r.With(s.authenticatedRateLimitMiddleware("secret-rotate", 5, time.Hour)).Post("/api/apps/{id}/secret", s.handleRotateSecret)
		r.Put("/api/apps/{id}/logo", s.handleUploadLogo)
		// Admin endpoints — access enforced by requireAdmin middleware.
		r.Route("/api/admin", func(r chi.Router) {
			r.Use(s.requireAdmin)
			r.Get("/apps", s.handleListAdminApps)
			r.Post("/apps/{id}/approve", s.handleApproveApp)
			r.Post("/apps/{id}/reject", s.handleRejectApp)
			// User management
			r.Get("/users", s.handleListAdminUsers)
			r.Delete("/users/{id}", s.handleDeleteAdminUser)
			// Handle blocklist
			r.Get("/handles/blocked", s.handleListBlockedHandles)
			r.Post("/handles/block", s.handleBlockHandle)
			r.Delete("/handles/{handle}", s.handleUnblockHandle)
			// Org logo management
			r.Get("/orgs", s.handleListAdminOrgs)
			r.Post("/orgs/{sid}/block-logo", s.handleBlockOrgLogo)
			r.Delete("/orgs/{sid}/block-logo", s.handleUnblockOrgLogo)
			// Report review queue
			r.Get("/reports", s.handleListAdminReports)
			r.Post("/reports/{id}/review", s.handleReviewReport)
			r.Post("/reports/{id}/dismiss", s.handleDismissReport)
		})
	})

	// Serve the embedded SvelteKit frontend for every path that isn't an API route.
	subFS, _ := fs.Sub(frontend.FS, "dist")
	// /config.js is generated at request time so the image stays deployment-agnostic.
	r.Get("/config.js", s.handleRuntimeConfig)
	r.Handle("/*", spaHandler(subFS))

	return r
}

// spaHandler serves static files from fsys with SPA fallback to index.html.
// SvelteKit's /_app/ assets are served with immutable cache headers because
// their filenames are content-hashed by the build tool.
func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	indexHTML, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		indexHTML = []byte(`<!doctype html><html><head><meta charset="utf-8"><title>SCID</title></head><body></body></html>`)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Aggressively cache SvelteKit's hashed asset bundles.
		if strings.HasPrefix(r.URL.Path, "/_app/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		// Fall back to index.html for any path that isn't a real file,
		// so that SvelteKit's client-side router handles the navigation.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "."
		}
		if _, err := fsys.Open(path); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(indexHTML)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// handleRuntimeConfig serves a small JS snippet that injects deployment-specific
// public environment variables into window.__SCID_PUBLIC_ENV__. The script is
// loaded by app.html so the Docker image stays fully generic.
func (s *Server) handleRuntimeConfig(w http.ResponseWriter, r *http.Request) {
	configJSON, err := json.Marshal(map[string]string{
		"PUBLIC_POCKET_ID_URL":      s.cfg.OIDCIssuerURL,
		"PUBLIC_OIDC_CLIENT_ID":     s.cfg.OIDCClientID,
		"PUBLIC_TURNSTILE_SITE_KEY": s.cfg.TurnstileSiteKey,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("window.__SCID_PUBLIC_ENV__=" + string(configJSON) + ";"))
}

// handleHealth returns a structured liveness/readiness check.
// Returns HTTP 200 while fully healthy or only Pocket ID is unreachable
// (degraded), and HTTP 503 when the database is unavailable.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Short deadline so slow upstream responses never stall health probes.
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	type depStatus struct {
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}

	deps := map[string]depStatus{}
	overall := "ok"
	code := http.StatusOK

	if err := s.store.Ping(ctx); err != nil {
		deps["database"] = depStatus{Status: "error", Message: err.Error()}
		overall = "degraded"
		code = http.StatusServiceUnavailable
	} else {
		deps["database"] = depStatus{Status: "ok"}
	}

	// Pocket ID being temporarily unreachable is reported as degraded (not 503)
	// so that companion health checks don't flap during Pocket ID restarts.
	if err := s.auth.Ping(ctx); err != nil {
		deps["pocket_id"] = depStatus{Status: "error", Message: err.Error()}
		if overall == "ok" {
			overall = "degraded"
		}
	} else {
		deps["pocket_id"] = depStatus{Status: "ok"}
	}

	writeJSON(w, code, map[string]any{
		"status": overall,
		"deps":   deps,
	})
}

// requireAdmin is a middleware that verifies the authenticated user is a member
// of the "admin" Pocket ID group. Must be used inside bearerAuthMiddleware.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := userFromContext(r.Context())
		ok, err := s.isUserInGroup(r, user.ID, "admin")
		if err != nil {
			slog.ErrorContext(r.Context(), "admin check failed", "user_id", user.ID, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if !ok {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// bearerAuthMiddleware validates the Authorization: Bearer token with Pocket ID
// and stores the resolved *pocketid.User in the request context.
func (s *Server) bearerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.resolveAuthenticatedUser(r)
		if err != nil {
			if !errors.Is(err, errMissingAuth) {
				slog.WarnContext(r.Context(), "session auth failed", "err", err)
			}
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
	})
}

// extractBearerToken parses the Authorization header and returns the token, or
// "" if the header is absent or malformed.
func extractBearerToken(r *http.Request) string {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if len(auth) <= len(prefix) {
		return ""
	}
	if auth[:len(prefix)] != prefix {
		return ""
	}
	return auth[len(prefix):]
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json response", "err", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// trustedRealIP is a middleware that sets r.RemoteAddr from X-Forwarded-For or
// X-Real-IP, but only when the direct peer is in cfg.TrustedProxies.
func (s *Server) trustedRealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.isTrustedProxy(r.RemoteAddr) {
			if ip := forwardedIP(r); ip != "" {
				r.RemoteAddr = ip
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) isTrustedProxy(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range s.cfg.TrustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func forwardedIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			xff = xff[:i]
		}
		if ip := net.ParseIP(strings.TrimSpace(xff)); ip != nil {
			return ip.String()
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		if ip := net.ParseIP(strings.TrimSpace(xrip)); ip != nil {
			return ip.String()
		}
	}
	return ""
}
