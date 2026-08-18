package auth

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/crypto"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/jwtx"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/password"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAuthEnv(t *testing.T) (*gorm.DB, *AuthService, *cache.Cache) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Role{}, &models.LoginLog{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewFromClient(client, "test")
	cipher, err := crypto.NewCipher("test-encryption-key-0123456789")
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	cfg := &config.Config{JWTExpiry: time.Hour, PasswordCost: 4}
	svc := NewAuthService(db, c, jwtx.NewManager("test-secret-0123456789abcdef", time.Hour), cipher, cfg, audit.NewWriter(db))
	return db, svc, c
}

func seedUser(t *testing.T, db *gorm.DB, email, pw string, active bool) models.User {
	t.Helper()
	hashed, err := password.Hash(pw, 4)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := models.User{Name: "Test User", Email: email, Password: hashed}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	// GORM fills default:true on the zero-value bool, so force inactive seeds.
	if !active {
		if err := db.Model(&u).Update("is_active", false).Error; err != nil {
			t.Fatalf("deactivate user: %v", err)
		}
	}
	return u
}

var testMeta = audit.Meta{UserID: uuid.New(), IP: "127.0.0.1", Method: "POST", Path: "/api/v1/auth/login"}

func TestLoginSuccessCreatesSessionAndClearsCounter(t *testing.T) {
	db, svc, c := newAuthEnv(t)
	seedUser(t, db, "a@b.com", "Secret123", true)
	ctx := context.Background()

	// A stale failure counter must be cleared by a successful login.
	if err := c.Set(ctx, "login:fail:a@b.com", "3", time.Hour); err != nil {
		t.Fatalf("seed counter: %v", err)
	}

	resp, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "Secret123"}, testMeta)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.AccessToken == "" || resp.TokenType != "Bearer" || resp.ExpiresIn <= 0 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if _, err := c.Get(ctx, "login:fail:a@b.com"); err == nil {
		t.Fatal("failure counter not cleared on success")
	}
	if _, err := c.Get(ctx, "session:"+resp.User.ID.String()); err != nil {
		t.Fatalf("session missing: %v", err)
	}
	var log models.LoginLog
	if err := db.First(&log, "email = ?", "a@b.com").Error; err != nil {
		t.Fatalf("login log missing: %v", err)
	}
	if log.Status != "success" {
		t.Fatalf("login log status = %s, want success", log.Status)
	}
}

func TestLoginRejectsWrongPasswordAndLogs(t *testing.T) {
	db, svc, _ := newAuthEnv(t)
	seedUser(t, db, "a@b.com", "Secret123", true)
	ctx := context.Background()

	if _, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "wrong"}, testMeta); err == nil {
		t.Fatal("wrong password must fail")
	}
	var log models.LoginLog
	if err := db.First(&log, "email = ?", "a@b.com").Error; err != nil {
		t.Fatalf("login log missing: %v", err)
	}
	if log.Status != "failed" {
		t.Fatalf("login log status = %s, want failed", log.Status)
	}
}

func TestLoginLocksAccountAfterFailures(t *testing.T) {
	db, svc, _ := newAuthEnv(t)
	seedUser(t, db, "a@b.com", "Secret123", true)
	ctx := context.Background()

	// Five failed attempts are allowed...
	for i := 0; i < maxLoginFailures; i++ {
		if _, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "wrong"}, testMeta); err == nil {
			t.Fatalf("attempt %d must fail", i+1)
		}
	}
	// ...the sixth is rejected with 429 even with the correct password.
	_, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "Secret123"}, testMeta)
	appErr := errors.As(err)
	if appErr == nil || appErr.Code != 429 {
		t.Fatalf("expected 429 after lockout, got %v", err)
	}
}

func TestLoginRejectsDisabledAccount(t *testing.T) {
	db, svc, _ := newAuthEnv(t)
	seedUser(t, db, "off@b.com", "Secret123", false)
	ctx := context.Background()

	_, err := svc.Login(ctx, LoginRequest{Email: "off@b.com", Password: "Secret123"}, testMeta)
	appErr := errors.As(err)
	if appErr == nil || appErr.Code != 403 {
		t.Fatalf("expected 403 for disabled account, got %v", err)
	}
}

func TestChangePasswordRotatesAndAudits(t *testing.T) {
	db, svc, _ := newAuthEnv(t)
	u := seedUser(t, db, "a@b.com", "Secret123", true)
	ctx := context.Background()

	if err := svc.ChangePassword(ctx, u.ID, ChangePasswordRequest{CurrentPassword: "nope", NewPassword: "NewSecret123"}, testMeta); err == nil {
		t.Fatal("wrong current password must fail")
	}
	if err := svc.ChangePassword(ctx, u.ID, ChangePasswordRequest{CurrentPassword: "Secret123", NewPassword: "NewSecret123"}, testMeta); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	var row models.User
	if err := db.First(&row, "id = ?", u.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !password.Verify(row.Password, "NewSecret123") {
		t.Fatal("password not rotated")
	}
	var auditRow models.AuditLog
	if err := db.First(&auditRow).Error; err != nil {
		t.Fatalf("audit entry missing: %v", err)
	}
	if auditRow.Action != "update" || len(auditRow.Before) == 0 {
		t.Fatalf("unexpected audit row: %+v", auditRow)
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	db, svc, c := newAuthEnv(t)
	seedUser(t, db, "a@b.com", "Secret123", true)
	ctx := context.Background()

	resp, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "Secret123"}, testMeta)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if err := svc.Logout(ctx, resp.User.ID, testMeta); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := c.Get(ctx, "session:"+resp.User.ID.String()); err == nil {
		t.Fatal("session still present after logout")
	}
}
