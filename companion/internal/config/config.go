package config

import (
	"fmt"
	"os"
)

// Config holds all configuration loaded from environment variables.
type Config struct {
	// ListenAddr is the address the HTTP server binds to.
	ListenAddr string

	// DatabasePath is the path to the SQLite database file.
	DatabasePath string

	// PocketIDInternalURL is the base URL of the Pocket ID service.
	PocketIDInternalURL string

	// PocketIDAdminAPIKey is the admin API key for Pocket ID.
	PocketIDAdminAPIKey string
}

// Load reads configuration from environment variables, applying defaults where
// appropriate. It returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:          getEnv("LISTEN_ADDR", ":8080"),
		DatabasePath:        getEnv("DATABASE_PATH", "/data/scid.db"),
		PocketIDInternalURL: getEnv("POCKET_ID_INTERNAL_URL", "http://pocket-id:3000"),
		PocketIDAdminAPIKey: os.Getenv("POCKET_ID_ADMIN_API_KEY"),
	}

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
