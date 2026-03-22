package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds all configuration loaded from environment variables.
type Config struct {
	// ListenAddr is the address the HTTP server binds to.
	ListenAddr string

	// DatabasePath is the path to the SQLite database file.
	DatabasePath string

	// PocketIDInternalURL is the base URL of the Pocket ID service.
	PocketIDInternalURL string

	// OIDCIssuerURL is the OIDC issuer URL used for auth discovery and token exchange.
	OIDCIssuerURL string

	// PocketIDAdminAPIKey is the admin API key for Pocket ID.
	PocketIDAdminAPIKey string

	// OIDCClientID is the frontend client ID used in the Pocket ID auth flow.
	OIDCClientID string

	// CORSAllowedOrigins is the list of allowed CORS origins for the companion API.
	// Configured via CORS_ALLOWED_ORIGINS (comma-separated). Defaults to production
	// origin plus localhost for dev convenience.
	CORSAllowedOrigins []string

	// SessionTTL is the maximum lifetime of a companion-managed login session.
	SessionTTL time.Duration

	// SessionCookieSecure controls whether the session cookie uses the Secure attribute.
	SessionCookieSecure bool

	// RequireAppApproval controls whether newly registered OIDC clients require
	// admin approval before they become active.  Defaults to true.
	RequireAppApproval bool

	// TurnstileSecretKey is the Cloudflare Turnstile secret used to validate
	// captcha responses server-side. If empty, captcha validation is skipped
	// (useful for local development).
	TurnstileSecretKey string
}

// Load reads configuration from environment variables, applying defaults where
// appropriate. It returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:          getEnv("LISTEN_ADDR", ":8080"),
		DatabasePath:        getEnv("DATABASE_PATH", "/data/scid.db"),
		PocketIDInternalURL: getEnv("POCKET_ID_INTERNAL_URL", "http://pocket-id:3000"),
		OIDCIssuerURL:       getEnv("POCKET_ID_ISSUER_URL", getEnv("PUBLIC_POCKET_ID_URL", getEnv("POCKET_ID_INTERNAL_URL", "http://pocket-id:3000"))),
		PocketIDAdminAPIKey: os.Getenv("POCKET_ID_ADMIN_API_KEY"),
		OIDCClientID:        getEnv("PUBLIC_OIDC_CLIENT_ID", "scid-frontend"),
		SessionCookieSecure: getEnv("SCID_COOKIE_SECURE", "true") != "false",
		RequireAppApproval:  getEnv("APP_REQUIRE_APPROVAL", "true") != "false",
		TurnstileSecretKey:  os.Getenv("TURNSTILE_SECRET_KEY"),
	}

	for _, o := range strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "https://scid.my,http://localhost:5173"), ",") {
		if s := strings.TrimSpace(o); s != "" {
			cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, s)
		}
	}

	sessionTTL, err := time.ParseDuration(getEnv("SCID_SESSION_TTL", "12h"))
	if err != nil {
		return nil, fmt.Errorf("parse SCID_SESSION_TTL: %w", err)
	}
	cfg.SessionTTL = sessionTTL

	if cfg.PocketIDAdminAPIKey == "" {
		return nil, fmt.Errorf("POCKET_ID_ADMIN_API_KEY environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
