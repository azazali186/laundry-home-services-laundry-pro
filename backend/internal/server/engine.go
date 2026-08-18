// Package server assembles the Hertz engine: middleware chain, route
// registration, permission seeding, swagger and WebSocket hub.
package server

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/jwtx"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/metrics"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/middleware"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/uuidx"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"
)

// Deps are the engine dependencies.
type Deps struct {
	Config   *config.Config
	DB       *gorm.DB
	Cache    *cache.Cache
	JWT      *jwtx.Manager
	Checker  middleware.PermissionChecker
	Bus      *bus.Bus
	Handlers *Handlers
}

// NewEngine builds the configured Hertz engine.
func NewEngine(deps Deps) *server.Hertz {
	h := server.New(
		server.WithHostPorts(":"+deps.Config.Port),
		server.WithMaxRequestBodySize(deps.Config.MaxBodyBytes),
		server.WithExitWaitTime(deps.Config.ShutdownTimeout),
	)

	// Global middleware chain.
	h.Use(middleware.Recovery())
	h.Use(RequestID())
	h.Use(middleware.RequestLog())
	h.Use(middleware.CORS(corsOrigins(deps.Config)))
	h.Use(middleware.SecurityHeaders(deps.Config.IsProduction()))
	h.Use(metrics.Middleware())
	h.Use(middleware.BodyLimit(deps.Config.MaxBodyBytes))

	// Infra endpoints (GET is required by the swagger UI and Prometheus
	// scrape protocol; all business endpoints are POST/PATCH/DELETE).
	h.GET("/swagger/*any", SwaggerHandler)
	h.GET("/metrics", func(ctx context.Context, c *app.RequestContext) {
		serveNetHTTP(c, metrics.Handler())
	})
	// Liveness and readiness probes (GET for orchestrators, POST for parity
	// with the API convention). The readiness check pings DB + Redis.
	h.GET("/api/v1/health", HealthHandler(deps.Config))
	h.POST("/api/v1/health", HealthHandler(deps.Config))
	h.GET("/api/v1/health/ready", ReadyHandler(HealthDeps{DB: deps.DB, Cache: deps.Cache}, deps.Config))
	h.POST("/api/v1/health/ready", ReadyHandler(HealthDeps{DB: deps.DB, Cache: deps.Cache}, deps.Config))

	// Public endpoints that must stay outside the auth middleware group.
	h.POST("/api/v1/auth/login", deps.Handlers.Auth.Login)
	// Public pricing catalog for the landing page (never gated by auth/RBAC
	// and never seeded as a permission). Per-IP rate limited and served from
	// a short-lived cache so it cannot hammer the database.
	h.POST("/api/v1/pricing",
		middleware.RateLimitScoped(deps.Cache, "pricing", max64(30, int64(deps.Config.RateLimitPerMin)/4)),
		deps.Handlers.Pricing.Public,
	)

	// Protected API group: auth + RBAC + rate limit. Sensitive or hot
	// routes get stricter per-route caps; everything else uses the global
	// RATE_LIMIT_PER_MIN default.
	api := h.Group("/api/v1")
	api.Use(middleware.Auth(deps.Cache, deps.JWT))
	api.Use(middleware.RateLimit(deps.Cache, int64(deps.Config.RateLimitPerMin), routeLimits(deps.Config.RateLimitPerMin)...))
	api.Use(middleware.RBAC(deps.Checker))
	RegisterRoutes(api, deps.Handlers)

	// WebSocket hub: authenticated via the Authorization header.
	MountWebSocket(h, deps.Cache, deps.JWT, deps.Bus)

	return h
}

// RequestID middleware assigns a traceable request id.
func RequestID() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		rid := string(c.Request.Header.Get("X-Request-Id"))
		if rid == "" {
			rid = uuidx.New().String()
		}
		middleware.SetRequestID(c, rid)
		response.WithRequestID(c, rid)
		c.Header("X-Request-Id", rid)
		c.Next(ctx)
	}
}

func corsOrigins(cfg *config.Config) string {
	if cfg.CORSOrigins != "" {
		return cfg.CORSOrigins
	}
	if cfg.IsProduction() {
		return "" // no browser origins allowed unless CORS_ORIGINS is set
	}
	return "*"
}

// routeLimits returns per-route caps derived from the global limit, so the
// caps scale automatically with RATE_LIMIT_PER_MIN instead of hardcoding
// absolute numbers. Longest-prefix matching applies in the middleware.
func routeLimits(global int) []middleware.RouteLimit {
	g := int64(global)
	return []middleware.RouteLimit{
		{Prefix: "/api/v1/auth/", Limit: max64(30, g/4)},    // password/security routes
		{Prefix: "/api/v1/bookings", Limit: max64(60, g/2)}, // creates sagas + outbox events
		{Prefix: "/api/v1/orders", Limit: max64(60, g/2)},   // status pipeline transitions
		{Prefix: "/api/v1/payments", Limit: max64(60, g/2)}, // money movement
		{Prefix: "/api/v1/invoices", Limit: max64(60, g/2)}, // billing docs
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
