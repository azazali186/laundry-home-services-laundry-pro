package middleware

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/jwtx"
	"github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// testSecret must satisfy the config minimum length (16 chars).
const testSecret = "test-secret-0123456789abcdef"

func newTestCache(t *testing.T) *cache.Cache {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return cache.NewFromClient(client, "test")
}

func okHandler(ctx context.Context, c *app.RequestContext) {
	c.String(http.StatusOK, "ok")
}

func TestAuthRejectsMissingToken(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(Auth(newTestCache(t), jwtx.NewManager(testSecret, time.Hour)))
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("missing token = %d, want 401", resp.Code)
	}
}

func TestAuthRejectsInvalidToken(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(Auth(newTestCache(t), jwtx.NewManager(testSecret, time.Hour)))
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil,
		ut.Header{Key: "Authorization", Value: "Bearer not-a-real-token"})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token = %d, want 401", resp.Code)
	}
}

func TestAuthAllowsValidSession(t *testing.T) {
	c := newTestCache(t)
	mgr := jwtx.NewManager(testSecret, time.Hour)
	userID := uuid.New()
	token, _, err := mgr.Issue(userID, "a@example.com", []string{"admin"})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	// Login flow stores the session fingerprint.
	if err := c.Set(context.Background(), "session:"+userID.String(), jwtx.SessionFingerprint(token, userID), time.Hour); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(Auth(c, mgr))
	h.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		id, err := UserID(c)
		if err != nil || id != userID {
			c.String(http.StatusInternalServerError, "identity lost")
			return
		}
		okHandler(ctx, c)
	})

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + token})
	if resp.Code != http.StatusOK {
		t.Fatalf("valid session = %d, want 200 (body: %s)", resp.Code, resp.Body.String())
	}
}

func TestAuthRejectsStaleSession(t *testing.T) {
	// Single-sign-on: a valid token whose session fingerprint does not match
	// the stored one (user logged in elsewhere) must be rejected.
	c := newTestCache(t)
	mgr := jwtx.NewManager(testSecret, time.Hour)
	userID := uuid.New()
	token, _, err := mgr.Issue(userID, "a@example.com", nil)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	// A *different* session is active: deliberately wrong fingerprint.
	if err := c.Set(context.Background(), "session:"+userID.String(), "some-other-session", time.Hour); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(Auth(c, mgr))
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + token})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("stale session = %d, want 401", resp.Code)
	}
}

func TestRBACDeniesWithoutPermission(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		SetIdentity(c, uuid.New(), "a@example.com", nil)
		c.Next(ctx)
	})
	h.Use(RBAC(allowChecker{allow: false}))
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("denied route = %d, want 403", resp.Code)
	}
}

func TestRBACAllowsWithPermission(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		SetIdentity(c, uuid.New(), "a@example.com", nil)
		c.Next(ctx)
	})
	h.Use(RBAC(allowChecker{allow: true}))
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("allowed route = %d, want 200", resp.Code)
	}
}

func TestBodyLimitRejectsOversizedBody(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(BodyLimit(16))
	h.POST("/ping", okHandler)

	big := bytes.Repeat([]byte("a"), 64)
	resp := ut.PerformRequest(h.Engine, http.MethodPost, "/ping", &ut.Body{Body: bytes.NewReader(big), Len: len(big)})
	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body = %d, want 413", resp.Code)
	}
}

func TestBodyLimitAllowsSmallBody(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(BodyLimit(1024))
	h.POST("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodPost, "/ping", &ut.Body{Body: strings.NewReader("tiny"), Len: 4})
	if resp.Code != http.StatusOK {
		t.Fatalf("small body = %d, want 200", resp.Code)
	}
}

func TestRecoveryReturns500Envelope(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(Recovery())
	h.GET("/boom", func(ctx context.Context, c *app.RequestContext) {
		panic("kaboom")
	})

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/boom", nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("panic route = %d, want 500", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"code":500`) {
		t.Fatalf("recovery body must be the error envelope, got %s", resp.Body.String())
	}
}

// allowChecker is a static PermissionChecker for RBAC tests.
type allowChecker struct{ allow bool }

func (a allowChecker) Check(_ context.Context, _ uuid.UUID, _, _ string) bool { return a.allow }
