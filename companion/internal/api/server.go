package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/lhw/scid/companion/internal/config"
	"github.com/lhw/scid/companion/internal/pocketid"
	"github.com/lhw/scid/companion/internal/rsi"
	"github.com/lhw/scid/companion/internal/store"
)

// Server wires together dependencies and the HTTP router.
type Server struct {
	cfg     *config.Config
	store   *store.Store
	pid     *pocketid.Client
	scraper *rsi.Scraper
	router  *chi.Mux
}

// New creates a configured Server ready to serve HTTP.
func New(cfg *config.Config, st *store.Store) *Server {
	s := &Server{
		cfg:     cfg,
		store:   st,
		pid:     pocketid.New(cfg.PocketIDInternalURL, cfg.PocketIDAdminAPIKey),
		scraper: rsi.New(),
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

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(prometheusMiddleware)

	// CORS — allow the frontend origin and localhost dev server.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://scid.my", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// TODO: add rate limiting middleware before public launch.

	// /metrics is served on the companion's internal port only and is not
	// proxied through Caddy — safe for Prometheus scraping within the stack.
	r.Mount("/metrics", prometheusHandler())
	r.Get("/api/health", s.handleHealth)
	// Status is intentionally public — returns {verified:false} when unauthenticated
	// so the home page can load without a login.
	r.Get("/api/verify/status", s.handleVerifyStatus)
	// Signup token — public endpoint that creates a 1-use Pocket ID registration token.
	// The frontend uses the token to redirect new users to Pocket ID's /signup page.
	r.Post("/api/auth/signup-token", s.handleSignupToken)
	// Org logo — serves cached org logos by SID (public, browser-cached).
	r.Get("/api/orgs/{sid}/logo", s.handleOrgLogo)
	// Public app directory — lists approved apps that have opted into the directory.
	r.Get("/api/apps/directory", s.handleListDirectoryApps)

	r.Group(func(r chi.Router) {
		r.Use(s.bearerAuthMiddleware)
		r.Post("/api/verify/start", s.handleVerifyStart)
		r.Post("/api/verify/confirm", s.handleVerifyConfirm)
		r.Post("/api/verify/refresh", s.handleVerifyRefresh)
		r.Post("/api/account/delete", s.handleDeleteAccount)
		r.Post("/api/apps", s.handleCreateApp)
		r.Get("/api/apps", s.handleListApps)
		r.Get("/api/apps/{id}", s.handleGetApp)
		r.Put("/api/apps/{id}", s.handleUpdateApp)
		r.Delete("/api/apps/{id}", s.handleDeleteApp)
		r.Post("/api/apps/{id}/secret", s.handleRotateSecret)
		r.Put("/api/apps/{id}/logo", s.handleUploadLogo)
		// Admin endpoints — require "admin" Pocket ID group (enforced in handler).
		r.Get("/api/admin/apps", s.handleListAdminApps)
		r.Post("/api/admin/apps/{id}/approve", s.handleApproveApp)
		r.Post("/api/admin/apps/{id}/reject", s.handleRejectApp)
	})

	return r
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
	if err := s.pid.Ping(ctx); err != nil {
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

// bearerAuthMiddleware validates the Authorization: Bearer token with Pocket ID
// and stores the resolved *pocketid.User in the request context.
func (s *Server) bearerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing or invalid Authorization header")
			return
		}

		user, err := s.pid.GetCurrentUser(r.Context(), token)
		if err != nil {
			slog.WarnContext(r.Context(), "bearer auth failed", "err", err)
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
