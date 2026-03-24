package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration loaded from environment variables.
type Config struct {
	// DatabaseURL is an optional PostgreSQL DSN used for production.
	// When empty, DatabasePath is used instead.
	DatabaseURL string

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

	// TurnstileSiteKey is the public Cloudflare Turnstile site key exposed to
	// the frontend at runtime. It is safe to inject into HTML responses.
	TurnstileSiteKey string

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

	// SMTP settings for outbound notification emails.
	// All fields are optional; when SMTPHost is empty, email sending is disabled.
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	// SMTPAdminEmail is the recipient address for admin notifications.
	SMTPAdminEmail string
}

// Load reads configuration from environment variables, applying defaults where
// appropriate. It returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		ListenAddr:          getEnv("LISTEN_ADDR", ":8080"),
		DatabasePath:        getEnv("DATABASE_PATH", "/data/scid.db"),
		PocketIDInternalURL: getEnv("POCKET_ID_INTERNAL_URL", "http://pocket-id:3000"),
		OIDCIssuerURL:       getEnv("POCKET_ID_ISSUER_URL", getEnv("PUBLIC_POCKET_ID_URL", getEnv("POCKET_ID_INTERNAL_URL", "http://pocket-id:3000"))),
		PocketIDAdminAPIKey: os.Getenv("POCKET_ID_ADMIN_API_KEY"),
		OIDCClientID:        getEnv("PUBLIC_OIDC_CLIENT_ID", "scid-frontend"),
		SessionCookieSecure: getEnv("SCID_COOKIE_SECURE", "true") != "false",
		RequireAppApproval:  getEnv("APP_REQUIRE_APPROVAL", "true") != "false",
		TurnstileSecretKey:  os.Getenv("TURNSTILE_SECRET_KEY"),
		TurnstileSiteKey:    getEnv("PUBLIC_TURNSTILE_SITE_KEY", ""),
		SMTPHost:            os.Getenv("SMTP_HOST"),
		SMTPPort:            smtpPort(getEnv("SMTP_PORT", "587")),
		SMTPUser:            os.Getenv("SMTP_USER"),
		SMTPPassword:        os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:            getEnv("SMTP_FROM", os.Getenv("SMTP_USER")),
		SMTPAdminEmail:      os.Getenv("SMTP_ADMIN_EMAIL"),
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

// DatabaseSource returns the configured database source string.
// PostgreSQL deployments use DatabaseURL; local development falls back to SQLite.
func (c *Config) DatabaseSource() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return c.DatabasePath
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func smtpPort(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 || n > 65535 {
		return 587
	}
	return n
}
