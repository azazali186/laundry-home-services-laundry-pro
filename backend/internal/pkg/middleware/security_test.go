package middleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(CORS("https://admin.example.com"))
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil,
		ut.Header{Key: "Origin", Value: "https://admin.example.com"})
	if resp.Code != http.StatusOK {
		t.Fatalf("allowed origin = %d, want 200", resp.Code)
	}
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Fatalf("ACAO = %q, want the allowed origin", got)
	}
	if got := resp.Header().Get("Access-Control-Allow-Methods"); got != "POST, PATCH, DELETE, OPTIONS" {
		t.Fatalf("ACAM = %q", got)
	}
}

func TestCORSRejectsDisallowedOrigin(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(CORS("https://admin.example.com"))
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil,
		ut.Header{Key: "Origin", Value: "https://evil.example.com"})
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed origin got ACAO = %q, want none", got)
	}
}

func TestCORSWildcardInDevelopment(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(CORS("*"))
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil,
		ut.Header{Key: "Origin", Value: "http://localhost:5173"})
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("wildcard ACAO = %q, want the origin echoed", got)
	}
}

func TestCORSPreflightShortCircuits(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(CORS("*"))
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodOptions, "/ping", nil,
		ut.Header{Key: "Origin", Value: "http://localhost:5173"})
	if resp.Code != http.StatusNoContent {
		t.Fatalf("preflight = %d, want 204", resp.Code)
	}
}

// clientIP pins the client identity so the per-IP rate-limit counter is shared
// across requests regardless of hertz's RemoteAddr resolution.
func clientIP() ut.Header { return ut.Header{Key: "X-Forwarded-For", Value: "10.0.0.1"} }

func TestRateLimitEnforcesLimit(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(RateLimit(newTestCache(t), 2))
	h.GET("/ping", okHandler)

	codes := []int{}
	for i := 0; i < 3; i++ {
		resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil, clientIP())
		codes = append(codes, resp.Code)
	}
	if codes[0] != http.StatusOK || codes[1] != http.StatusOK || codes[2] != http.StatusTooManyRequests {
		t.Fatalf("rate limit codes = %v, want [200 200 429]", codes)
	}
}

func TestRateLimitRouteCap(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(RateLimit(newTestCache(t), 100, RouteLimit{Prefix: "/auth", Limit: 1}))
	h.GET("/auth/login", okHandler)
	h.GET("/ping", okHandler)

	// The counter is shared per client IP; the path only changes the threshold.
	// /auth/login is capped at 1 per minute even though the global is 100.
	if resp := ut.PerformRequest(h.Engine, http.MethodGet, "/auth/login", nil, clientIP()); resp.Code != http.StatusOK {
		t.Fatalf("first /auth/login = %d, want 200", resp.Code)
	}
	if resp := ut.PerformRequest(h.Engine, http.MethodGet, "/auth/login", nil, clientIP()); resp.Code != http.StatusTooManyRequests {
		t.Fatalf("second /auth/login = %d, want 429", resp.Code)
	}
	// A different route uses the same counter against the global limit (100),
	// so the shared count of 2 is still allowed.
	if resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil, clientIP()); resp.Code != http.StatusOK {
		t.Fatalf("/ping = %d, want 200", resp.Code)
	}
}

func TestRateLimitDegradesOpenOnCacheFailure(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	broken := cache.NewFromClient(client, "test")
	mr.Close() // cache now unreachable

	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(RateLimit(broken, 1))
	h.GET("/ping", okHandler)

	for i := 0; i < 3; i++ {
		resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil, clientIP())
		if resp.Code != http.StatusOK {
			t.Fatalf("cache failure request %d = %d, want 200 (degrade open)", i, resp.Code)
		}
	}
}

func TestContextAccessors(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	userID := uuid.New()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		SetIdentity(c, userID, "u@example.com", []string{"admin", "manager"})
		SetRequestID(c, "req-123")
		c.Next(ctx)
	})
	h.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		if got := Email(c); got != "u@example.com" {
			c.String(http.StatusInternalServerError, "email mismatch")
			return
		}
		if got := RequestID(c); got != "req-123" {
			c.String(http.StatusInternalServerError, "request id mismatch")
			return
		}
		roles := Roles(c)
		if len(roles) != 2 || roles[0] != "admin" || roles[1] != "manager" {
			c.String(http.StatusInternalServerError, "roles mismatch")
			return
		}
		okHandler(ctx, c)
	})

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("context accessors = %d, want 200 (body: %s)", resp.Code, resp.Body.String())
	}
}

func TestRequestLogRuns(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(RequestLog())
	h.GET("/ping", okHandler)

	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("request log = %d, want 200", resp.Code)
	}
}

func TestRateLimitScopedIsolation(t *testing.T) {
	// Two scoped limiters (public route vs API group) must not share a per-IP
	// counter: each scope gets its own 1/min budget for the same client.
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.POST("/pricing", RateLimitScoped(newTestCache(t), "pricing", 1), okHandler)
	api := h.Group("/api/v1")
	api.Use(RateLimitScoped(newTestCache(t), "api", 1))
	api.GET("/health", okHandler)

	// First call in each scope passes; the api scope passes even after the
	// pricing budget was consumed, proving the counters are independent.
	if resp := ut.PerformRequest(h.Engine, http.MethodPost, "/pricing", nil, clientIP()); resp.Code != http.StatusOK {
		t.Fatalf("pricing #1 = %d, want 200", resp.Code)
	}
	if resp := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/health", nil, clientIP()); resp.Code != http.StatusOK {
		t.Fatalf("api #1 = %d, want 200 (isolated counter)", resp.Code)
	}
	if resp := ut.PerformRequest(h.Engine, http.MethodPost, "/pricing", nil, clientIP()); resp.Code != http.StatusTooManyRequests {
		t.Fatalf("pricing #2 = %d, want 429 (own budget)", resp.Code)
	}
	if resp := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/health", nil, clientIP()); resp.Code != http.StatusTooManyRequests {
		t.Fatalf("api #2 = %d, want 429 (own budget)", resp.Code)
	}
}

func TestRateLimitUserScoped(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	fixedUser := uuid.New()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		SetIdentity(c, fixedUser, "a@example.com", nil)
		c.Next(ctx)
	})
	h.Use(RateLimit(newTestCache(t), 1))
	h.GET("/ping", okHandler)

	if resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil, clientIP()); resp.Code != http.StatusOK {
		t.Fatalf("first user request = %d, want 200", resp.Code)
	}
	if resp := ut.PerformRequest(h.Engine, http.MethodGet, "/ping", nil, clientIP()); resp.Code != http.StatusTooManyRequests {
		t.Fatalf("second user request = %d, want 429", resp.Code)
	}
}
