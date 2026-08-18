// Package config loads and validates application configuration from the
// environment. All values are snake_case friendly and documented in .env.example.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the backend.
type Config struct {
	Port            string
	Env             string
	LogLevel        string
	LogFormat       string
	DatabaseURL     string
	RedisURL        string
	RedisPrefix     string
	NATSURL         string
	NATSEnabled     bool
	JWTSecret       string
	JWTExpiry       time.Duration
	EncryptionKey   string
	PasswordCost    int
	AdminEmail      string
	AdminPassword   string
	AdminName       string
	CORSOrigins     string
	ShutdownTimeout time.Duration
	OutboxPoll      time.Duration
	OutboxBatch     int
	OutboxMaxRetry  int
	RateLimitPerMin int
	MaxBodyBytes    int
}

// Known insecure defaults that must never be accepted in production.
var devDefaults = []string{
	"dev-secret-change-me",
	"dev-encryption-key-change-me",
	"change-me-to-a-long-random-secret-at-least-32-chars",
	"change-me-32-bytes-secret-key-xxxx",
	"Admin@12345",
}

// Load reads configuration from environment variables, applying defaults.
// Log level defaults to debug in development and info elsewhere.
func Load() (*Config, error) {
	env := getEnv("ENV", "development")
	defaultLogLevel := "info"
	if env == "development" {
		defaultLogLevel = "debug"
	}
	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		Env:             env,
		LogLevel:        getEnv("LOG_LEVEL", defaultLogLevel),
		LogFormat:       getEnv("LOG_FORMAT", "text"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://aeroxe:secret@localhost:5433/laundry_pro?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisPrefix:     getEnv("REDIS_PREFIX", "laundry-pro"),
		NATSURL:         getEnv("NATS_URL", "nats://localhost:4222"),
		NATSEnabled:     getBoolEnv("NATS_ENABLED", false),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiry:       getDurationEnv("JWT_EXPIRY", 24*time.Hour),
		EncryptionKey:   getEnv("ENCRYPTION_KEY", "dev-encryption-key-change-me"),
		PasswordCost:    getIntEnv("PASSWORD_COST", 12),
		AdminEmail:      getEnv("ADMIN_EMAIL", "admin@laundrypro.local"),
		AdminPassword:   getEnv("ADMIN_PASSWORD", "Admin@12345"),
		AdminName:       getEnv("ADMIN_NAME", "System Administrator"),
		CORSOrigins:     getEnv("CORS_ORIGINS", ""),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 15*time.Second),
		OutboxPoll:      getDurationEnv("OUTBOX_POLL_INTERVAL", 2*time.Second),
		OutboxBatch:     getIntEnv("OUTBOX_BATCH_SIZE", 50),
		OutboxMaxRetry:  getIntEnv("OUTBOX_MAX_RETRIES", 5),
		RateLimitPerMin: getIntEnv("RATE_LIMIT_PER_MIN", 120),
		MaxBodyBytes:    getIntEnv("MAX_BODY_BYTES", 5<<20),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func containsDevDefault(v string) bool {
	for _, d := range devDefaults {
		if v == d {
			return true
		}
	}
	return false
}

func (c *Config) validate() error {
	if c.OutboxBatch < 1 {
		return fmt.Errorf("OUTBOX_BATCH_SIZE must be >= 1")
	}
	if c.RateLimitPerMin < 1 {
		return fmt.Errorf("RATE_LIMIT_PER_MIN must be >= 1")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive")
	}
	if c.LogFormat != "text" && c.LogFormat != "json" {
		return fmt.Errorf("LOG_FORMAT must be text or json")
	}

	// Common minimums: keys must be strong enough to sign/encrypt safely.
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if len(c.EncryptionKey) < 16 {
		return fmt.Errorf("ENCRYPTION_KEY must be at least 16 characters")
	}

	if c.IsProduction() {
		// Fail fast instead of silently running with known credentials.
		if containsDevDefault(c.JWTSecret) || len(c.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be a unique value of at least 32 characters in production")
		}
		if containsDevDefault(c.EncryptionKey) || len(c.EncryptionKey) < 32 {
			return fmt.Errorf("ENCRYPTION_KEY must be a unique value of at least 32 characters in production")
		}
		if containsDevDefault(c.AdminPassword) || len(c.AdminPassword) < 12 {
			return fmt.Errorf("ADMIN_PASSWORD must be a unique value of at least 12 characters in production")
		}
		if c.PasswordCost < 12 {
			return fmt.Errorf("PASSWORD_COST must be >= 12 in production")
		}
		if strings.Contains(c.DatabaseURL, "@localhost") {
			return fmt.Errorf("DATABASE_URL must not point to localhost in production")
		}
	}
	return nil
}

// IsDevelopment reports whether the app runs in a development environment.
func (c *Config) IsDevelopment() bool { return c.Env == "development" }

// IsProduction reports whether the app runs in a production environment.
func (c *Config) IsProduction() bool { return c.Env == "production" }

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	v := strings.ToLower(getEnv(key, ""))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getIntEnv(key string, fallback int) int {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
