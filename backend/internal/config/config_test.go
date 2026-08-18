package config

import (
	"strings"
	"testing"
)

// strongProd sets every secret to a value that passes production validation.
func strongProd(t *testing.T) {
	t.Helper()
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", strings.Repeat("j", 32))
	t.Setenv("ENCRYPTION_KEY", strings.Repeat("e", 32))
	t.Setenv("ADMIN_PASSWORD", strings.Repeat("p", 12))
	t.Setenv("PASSWORD_COST", "12")
	t.Setenv("DATABASE_URL", "postgres://prod:secret@db:5432/laundry_pro?sslmode=disable")
}

func TestLoadDevelopmentDefaults(t *testing.T) {
	t.Setenv("ENV", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsDevelopment() || cfg.Port != "8080" {
		t.Fatalf("unexpected dev config: %+v", cfg)
	}
	if cfg.MaxBodyBytes != 5<<20 {
		t.Fatalf("default body limit = %d, want %d", cfg.MaxBodyBytes, 5<<20)
	}
}

func TestProductionRejectsDevSecrets(t *testing.T) {
	strongProd(t)
	// JWT_SECRET still the dev default.
	t.Setenv("JWT_SECRET", "dev-secret-change-me")
	if _, err := Load(); err == nil {
		t.Fatal("production must reject the dev JWT secret")
	}
}

func TestProductionRejectsShortJWTSecret(t *testing.T) {
	strongProd(t)
	t.Setenv("JWT_SECRET", "short")
	if _, err := Load(); err == nil {
		t.Fatal("production must reject a short JWT secret")
	}
}

func TestProductionRejectsWeakAdminPassword(t *testing.T) {
	strongProd(t)
	t.Setenv("ADMIN_PASSWORD", "Admin@12345") // known dev default
	if _, err := Load(); err == nil {
		t.Fatal("production must reject the dev admin password")
	}
}

func TestProductionRejectsLowPasswordCost(t *testing.T) {
	strongProd(t)
	t.Setenv("PASSWORD_COST", "4")
	if _, err := Load(); err == nil {
		t.Fatal("production must require PASSWORD_COST >= 12")
	}
}

func TestProductionRejectsLocalhostDatabaseURL(t *testing.T) {
	strongProd(t)
	t.Setenv("DATABASE_URL", "postgres://aeroxe:secret@localhost:5433/laundry_pro?sslmode=disable")
	if _, err := Load(); err == nil {
		t.Fatal("production must reject localhost DATABASE_URL")
	}
}

func TestProductionAcceptsStrongSecrets(t *testing.T) {
	strongProd(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsProduction() {
		t.Fatal("expected production env")
	}
}

func TestInvalidLogFormatRejected(t *testing.T) {
	t.Setenv("LOG_FORMAT", "yaml")
	if _, err := Load(); err == nil {
		t.Fatal("invalid LOG_FORMAT must fail")
	}
}

func TestInvalidOutboxBatchRejected(t *testing.T) {
	t.Setenv("OUTBOX_BATCH_SIZE", "0")
	if _, err := Load(); err == nil {
		t.Fatal("OUTBOX_BATCH_SIZE < 1 must fail")
	}
}
