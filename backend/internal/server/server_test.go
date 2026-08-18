package server

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/crypto"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/password"
	"github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/hertz/pkg/app"
	hertzserver "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openPermDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Permission{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestIsExcludedRoute(t *testing.T) {
	excluded := []string{
		"/api/v1/auth/login", "/api/v1/auth/change-password",
		"/swagger/index.html", "/swagger/doc.json",
		"/metrics", "/api/v1/health", "/api/v1/health/ready", "/ws",
		"/api/v1/pricing", // public catalog: never an RBAC permission
	}
	for _, p := range excluded {
		if !IsExcludedRoute(p) {
			t.Errorf("IsExcludedRoute(%q) = false, want true", p)
		}
	}
	kept := []string{"/api/v1/bookings", "/api/v1/orders/list", "/api/v1/settings", "/api/v1/combos", "/api/v1/combos/list"}
	for _, p := range kept {
		if IsExcludedRoute(p) {
			t.Errorf("IsExcludedRoute(%q) = true, want false", p)
		}
	}
}

func TestFormatRouteName(t *testing.T) {
	cases := map[string]string{
		"/api/v1/bookings/list": "Bookings List",
		"/api/v1/users":         "Users",
		"/api/v1/settings":      "Settings",
	}
	for path, want := range cases {
		if got := formatRouteName(path); got != want {
			t.Errorf("formatRouteName(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestUpsertPermissionCreatesAndUpdates(t *testing.T) {
	db := openPermDB(t)
	ctx := context.Background()
	route := RouteInfo{Name: "Bookings", URL: "/api/v1/bookings", Guard: "API", Method: "POST"}

	if err := upsertPermission(ctx, db, route); err != nil {
		t.Fatalf("upsert create: %v", err)
	}
	var row models.Permission
	if err := db.First(&row, "route = ?", "POST /api/v1/bookings").Error; err != nil {
		t.Fatalf("permission missing: %v", err)
	}
	if row.Method != "POST" || row.Service != "api-gateway" || row.Guard != "API" {
		t.Fatalf("unexpected permission row: %+v", row)
	}

	// Renaming the route updates the existing row instead of duplicating it.
	route.Name = "Bookings List"
	if err := upsertPermission(ctx, db, route); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	var count int64
	db.Model(&models.Permission{}).Count(&count)
	if count != 1 {
		t.Fatalf("rows = %d, want 1 (no duplicates)", count)
	}
	var updated models.Permission
	db.First(&updated, "route = ?", "POST /api/v1/bookings")
	if updated.Name != "Bookings List" {
		t.Fatalf("name not updated: %s", updated.Name)
	}
}

func TestSeedBootstrapCreatesRolesAndAdmin(t *testing.T) {
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Role{}, &models.Permission{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cipher, err := crypto.NewCipher("test-encryption-key-0123456789")
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	cfg := &config.Config{
		AdminEmail: "admin@x.local", AdminPassword: "AdminPass123",
		AdminName: "System Admin", PasswordCost: 4,
	}
	ctx := context.Background()

	if err := SeedBootstrap(ctx, db, cfg, cipher); err != nil {
		t.Fatalf("SeedBootstrap: %v", err)
	}
	// Idempotent: a second run must not duplicate roles or the admin.
	if err := SeedBootstrap(ctx, db, cfg, cipher); err != nil {
		t.Fatalf("SeedBootstrap second run: %v", err)
	}

	var roleCount int64
	db.Model(&models.Role{}).Count(&roleCount)
	if roleCount != 4 {
		t.Fatalf("roles = %d, want 4", roleCount)
	}

	var admin models.User
	if err := db.Preload("Roles").First(&admin, "email = ?", cfg.AdminEmail).Error; err != nil {
		t.Fatalf("admin missing: %v", err)
	}
	if !password.Verify(admin.Password, cfg.AdminPassword) {
		t.Fatal("admin password does not verify")
	}
	hasSuper := false
	for _, r := range admin.Roles {
		if r.Code == RoleSuperAdmin {
			hasSuper = true
		}
	}
	if !hasSuper {
		t.Fatalf("admin lacks super_admin role: %+v", admin.Roles)
	}

	// Wildcard permission must be attached to the super admin role.
	var superRole models.Role
	if err := db.Preload("Permissions").First(&superRole, "code = ?", RoleSuperAdmin).Error; err != nil {
		t.Fatalf("super role missing: %v", err)
	}
	hasWildcard := false
	for _, p := range superRole.Permissions {
		if p.Route == "*" {
			hasWildcard = true
		}
	}
	if !hasWildcard {
		t.Fatal("super admin lacks wildcard permission")
	}
}

func TestSeedPermissionsRegistersOnlyNonExcludedRoutes(t *testing.T) {
	db := openPermDB(t)
	ctx := context.Background()

	h := hertzserver.New(hertzserver.WithHostPorts("127.0.0.1:0"))
	h.POST("/api/v1/bookings", func(c context.Context, ctx *app.RequestContext) {})
	h.GET("/api/v1/orders/list", func(c context.Context, ctx *app.RequestContext) {})
	h.POST("/api/v1/combos/list", func(c context.Context, ctx *app.RequestContext) {})
	h.POST("/api/v1/auth/login", func(c context.Context, ctx *app.RequestContext) {}) // excluded
	h.GET("/api/v1/health", func(c context.Context, ctx *app.RequestContext) {})      // excluded
	h.POST("/api/v1/pricing", func(c context.Context, ctx *app.RequestContext) {})    // excluded

	if err := SeedPermissions(ctx, db, h); err != nil {
		t.Fatalf("SeedPermissions: %v", err)
	}

	var rows []models.Permission
	if err := db.Order("route ASC").Find(&rows).Error; err != nil {
		t.Fatalf("load permissions: %v", err)
	}
	routes := map[string]bool{}
	for _, r := range rows {
		routes[r.Route] = true
	}
	if !routes["POST /api/v1/bookings"] {
		t.Fatal("bookings route not seeded as permission")
	}
	if !routes["GET /api/v1/orders/list"] {
		t.Fatal("orders list route not seeded as permission")
	}
	if !routes["POST /api/v1/combos/list"] {
		t.Fatal("combos list route not seeded as permission")
	}
	if routes["POST /api/v1/auth/login"] {
		t.Fatal("excluded auth/login route must not become a permission")
	}
	if routes["GET /api/v1/health"] {
		t.Fatal("excluded health route must not become a permission")
	}
	if routes["POST /api/v1/pricing"] {
		t.Fatal("public pricing route must not become a permission")
	}
}

func TestHealthAndReadyHandlers(t *testing.T) {
	db := openPermDB(t)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewFromClient(client, "test")
	cfg := &config.Config{Env: "test"}

	// Liveness always answers 200.
	h := hertzserver.New(hertzserver.WithHostPorts("127.0.0.1:0"))
	h.GET("/api/v1/health", HealthHandler(cfg))
	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/health", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("liveness = %d, want 200", resp.Code)
	}

	// Readiness reports both dependencies ok.
	r := hertzserver.New(hertzserver.WithHostPorts("127.0.0.1:0"))
	r.GET("/api/v1/health/ready", ReadyHandler(HealthDeps{DB: db, Cache: c}, cfg))
	resp = ut.PerformRequest(r.Engine, http.MethodGet, "/api/v1/health/ready", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("readiness = %d, want 200", resp.Code)
	}
	if got := string(resp.Body.Bytes()); !strings.Contains(got, `"database":"ok"`) || !strings.Contains(got, `"cache":"ok"`) {
		t.Fatalf("readiness body missing ok checks: %s", got)
	}
}

func TestReadyHandlerDegradesWhenDependenciesDown(t *testing.T) {
	cfg := &config.Config{Env: "test"}

	// Cache down -> 503 and the body names the failing dependency.
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewFromClient(client, "test")
	mr.Close()

	h := hertzserver.New(hertzserver.WithHostPorts("127.0.0.1:0"))
	h.GET("/api/v1/health/ready", ReadyHandler(HealthDeps{Cache: c}, cfg))
	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/health/ready", nil)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness (cache down) = %d, want 503", resp.Code)
	}

	// DB down -> 503 and the body names the database.
	db := openPermDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sqlDB: %v", err)
	}
	_ = sqlDB.Close()

	r := hertzserver.New(hertzserver.WithHostPorts("127.0.0.1:0"))
	r.GET("/api/v1/health/ready", ReadyHandler(HealthDeps{DB: db}, cfg))
	resp = ut.PerformRequest(r.Engine, http.MethodGet, "/api/v1/health/ready", nil)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness (db down) = %d, want 503", resp.Code)
	}
	if got := string(resp.Body.Bytes()); !strings.Contains(got, "database") {
		t.Fatalf("readiness body missing database failure: %s", got)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	// A missing request id is generated and echoed back.
	h := hertzserver.New(hertzserver.WithHostPorts("127.0.0.1:0"))
	h.Use(RequestID())
	h.GET("/x", func(ctx context.Context, c *app.RequestContext) {
		c.SetStatusCode(http.StatusNoContent)
	})
	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/x", nil)
	got := string(resp.Header().Get("X-Request-Id"))
	if got == "" {
		t.Fatal("X-Request-Id not set")
	}

	// An inbound request id is preserved, not regenerated.
	const sent = "trace-abc-123"
	resp = ut.PerformRequest(h.Engine, http.MethodGet, "/x", nil,
		ut.Header{Key: "X-Request-Id", Value: sent})
	if got := string(resp.Header().Get("X-Request-Id")); got != sent {
		t.Fatalf("X-Request-Id = %q, want %q (preserved)", got, sent)
	}
}

func TestCorsOriginsPolicy(t *testing.T) {
	// Explicit allow-list wins in every environment.
	allow := &config.Config{Env: "development", CORSOrigins: "https://app.example.com"}
	if got := corsOrigins(allow); got != "https://app.example.com" {
		t.Fatalf("explicit origins = %q", got)
	}
	// Dev falls back to wildcard; production denies browser origins.
	dev := &config.Config{Env: "development"}
	if got := corsOrigins(dev); got != "*" {
		t.Fatalf("dev origins = %q, want *", got)
	}
	prod := &config.Config{Env: "production"}
	if got := corsOrigins(prod); got != "" {
		t.Fatalf("prod origins = %q, want empty (deny all)", got)
	}
}

func TestRouteLimitsScaleWithGlobal(t *testing.T) {
	// Default 120/min: auth routes cap at 30, bookings/orders at 60.
	ticks := routeLimits(120)
	byPrefix := map[string]int64{}
	for _, r := range ticks {
		byPrefix[r.Prefix] = r.Limit
	}
	if byPrefix["/api/v1/auth/"] != 30 || byPrefix["/api/v1/bookings"] != 60 || byPrefix["/api/v1/payments"] != 60 {
		t.Fatalf("unexpected limits at 120: %+v", byPrefix)
	}
	// Generous global raises the per-route caps proportionally.
	ticks = routeLimits(400)
	byPrefix = map[string]int64{}
	for _, r := range ticks {
		byPrefix[r.Prefix] = r.Limit
	}
	if byPrefix["/api/v1/auth/"] != 100 || byPrefix["/api/v1/bookings"] != 200 {
		t.Fatalf("unexpected limits at 400: %+v", byPrefix)
	}
	// Floor applies when the global limit is tiny.
	ticks = routeLimits(10)
	for _, r := range ticks {
		if r.Prefix == "/api/v1/auth/" && r.Limit != 30 {
			t.Fatalf("auth floor not applied: %d", r.Limit)
		}
	}
}

func TestServeNetHTTPBridge(t *testing.T) {
	h := hertzserver.New(hertzserver.WithHostPorts("127.0.0.1:0"))
	h.GET("/bridge", func(ctx context.Context, c *app.RequestContext) {
		serveNetHTTP(c, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte("teapot"))
		}))
	})
	resp := ut.PerformRequest(h.Engine, http.MethodGet, "/bridge", nil)
	if resp.Code != http.StatusTeapot {
		t.Fatalf("bridge status = %d, want 418", resp.Code)
	}
	if got := string(resp.Body.Bytes()); got != "teapot" {
		t.Fatalf("bridge body = %q", got)
	}
	if ct := string(resp.Header().Get("Content-Type")); ct != "text/plain" {
		t.Fatalf("bridge content-type = %q", ct)
	}
}
