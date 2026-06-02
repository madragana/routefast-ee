// Package config loads runtime configuration for the lipd-server from the
// environment. All settings have sensible defaults for local development.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all runtime settings for the credential server.
type Config struct {
	// ListenAddr is the host:port the mTLS API server binds to.
	ListenAddr string

	// DatabaseURL is the YugabyteDB (PostgreSQL wire) connection string.
	DatabaseURL string

	// TLS material for the mTLS endpoint.
	TLSCertFile     string
	TLSKeyFile      string
	TLSClientCAFile string

	// Token / key lifecycle settings.
	TokenTTL       time.Duration // bearer token validity (default 7d)
	KeyRotationTTL time.Duration // ed25519 keypair validity (default 7d)

	// Operational tuning.
	MaxDBConns int32
}

// Load reads configuration from environment variables, applying defaults.
func Load() (*Config, error) {
	c := &Config{
		ListenAddr:      env("LISTEN_ADDR", ":8443"),
		DatabaseURL:     env("DATABASE_URL", "postgres://yugabyte:yugabyte@localhost:5433/yugabyte"),
		TLSCertFile:     env("TLS_CERT_FILE", "./tls/server.crt"),
		TLSKeyFile:      env("TLS_KEY_FILE", "./tls/server.key"),
		TLSClientCAFile: env("TLS_CLIENT_CA_FILE", "./tls/ca.crt"),
		TokenTTL:        7 * 24 * time.Hour,
		KeyRotationTTL:  7 * 24 * time.Hour,
		MaxDBConns:      16,
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL must be set")
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
