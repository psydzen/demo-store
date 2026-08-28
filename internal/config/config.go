// Package config reads the application settings from the environment.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds everything the application needs to start.
type Config struct {
	// Addr is the TCP address the HTTP server listens on.
	Addr string
	// DatabaseURL is the Postgres connection string.
	DatabaseURL string
	// AdminToken, when set, replaces the seeded admin token on startup.
	AdminToken string
	// SessionLifetime is how long a login stays valid.
	SessionLifetime time.Duration
}

// minAdminTokenLen matches the shortest token the generator produces, so a
// hand-written override cannot be weaker than a generated one.
const minAdminTokenLen = 16

// Load reads the configuration from the environment and validates it.
func Load() (Config, error) {
	cfg := Config{
		Addr:            env("ADDR", ":8080"),
		DatabaseURL:     env("DATABASE_URL", ""),
		AdminToken:      strings.TrimSpace(os.Getenv("ADMIN_TOKEN")),
		SessionLifetime: 12 * time.Hour,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is not set")
	}
	if cfg.AdminToken != "" && len(cfg.AdminToken) < minAdminTokenLen {
		return Config{}, fmt.Errorf("ADMIN_TOKEN must be at least %d characters", minAdminTokenLen)
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
