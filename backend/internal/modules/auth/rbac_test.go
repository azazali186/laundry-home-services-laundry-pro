package auth

import (
	"context"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite" // pure-Go driver, no CGO required
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newRbacEnv(t *testing.T) (*gorm.DB, *RbacService, *miniredis.Miniredis) {
	t.Helper()
	// Unique DSN per test: with cache=shared, a bare :memory: DSN would be
	// shared across every test in the process.
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Role{}, &models.Permission{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewFromClient(client, "laundry-pro")
	return db, NewRbacService(db, c), mr
}

// seedRBAC creates a user with one role and the given permission routes.
func seedRBAC(t *testing.T, db *gorm.DB, userID uuid.UUID, routes ...string) {
	t.Helper()
	role := models.Role{Name: "agent", Code: "AGENT"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	for _, r := range routes {
		parts := splitRoute(r)
		perm := models.Permission{
			Name: r, Route: r, Path: parts[1], Method: parts[0], Service: "api-gateway",
		}
		if err := db.Create(&perm).Error; err != nil {
			t.Fatalf("create permission %s: %v", r, err)
		}
		role.Permissions = append(role.Permissions, perm)
	}
	if err := db.Save(&role).Error; err != nil {
		t.Fatalf("save role perms: %v", err)
	}
	user := models.User{
		Base: models.Base{ID: userID}, Name: "Tester", Email: "tester@example.com",
		Password: "x", IsActive: true, Roles: []models.Role{role},
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func splitRoute(route string) [2]string {
	for i := 0; i < len(route); i++ {
		if route[i] == ' ' {
			return [2]string{route[:i], route[i+1:]}
		}
	}
	return [2]string{"*", route}
}

func TestCheckAllowsGrantedRoute(t *testing.T) {
	db, svc, _ := newRbacEnv(t)
	uid := uuid.New()
	seedRBAC(t, db, uid, "POST /api/v1/orders", "GET /api/v1/orders")

	ctx := context.Background()
	if !svc.Check(ctx, uid, "POST", "/api/v1/orders") {
		t.Fatal("granted POST route should pass")
	}
	if !svc.Check(ctx, uid, "GET", "/api/v1/orders") {
		t.Fatal("granted GET route should pass")
	}
	if svc.Check(ctx, uid, "DELETE", "/api/v1/orders") {
		t.Fatal("ungranted method should be denied")
	}
	if svc.Check(ctx, uid, "POST", "/api/v1/orders/123") {
		t.Fatal("unrelated path should be denied")
	}
}

func TestCheckUnknownUserDenied(t *testing.T) {
	db, svc, _ := newRbacEnv(t)
	uid := uuid.New()
	seedRBAC(t, db, uid, "POST /api/v1/orders")

	ctx := context.Background()
	if svc.Check(ctx, uuid.New(), "POST", "/api/v1/orders") {
		t.Fatal("unknown user must be denied")
	}
}

func TestCheckInactiveUserDenied(t *testing.T) {
	db, svc, _ := newRbacEnv(t)
	uid := uuid.New()
	seedRBAC(t, db, uid, "POST /api/v1/orders")
	if err := db.Model(&models.User{}).Where("id = ?", uid).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	ctx := context.Background()
	if svc.Check(ctx, uid, "POST", "/api/v1/orders") {
		t.Fatal("inactive user must be denied")
	}
}

func TestInvalidateClearsCache(t *testing.T) {
	db, svc, mr := newRbacEnv(t)
	uid := uuid.New()
	seedRBAC(t, db, uid, "POST /api/v1/orders")

	ctx := context.Background()
	if !svc.Check(ctx, uid, "POST", "/api/v1/orders") {
		t.Fatal("first check should pass and populate cache")
	}
	// Remove the permission directly in the DB (bypassing the service).
	if err := db.Where("route = ?", "POST /api/v1/orders").Delete(&models.Permission{}).Error; err != nil {
		t.Fatalf("delete perm: %v", err)
	}
	// Without invalidation the cached snapshot still grants access.
	if !svc.Check(ctx, uid, "POST", "/api/v1/orders") {
		t.Fatal("cached snapshot should still grant (until invalidated)")
	}
	svc.Invalidate(ctx)
	if _, err := mr.Get("laundry-pro:rbac:version"); err != nil {
		t.Fatalf("version key not bumped: %v", err)
	}
	if svc.Check(ctx, uid, "POST", "/api/v1/orders") {
		t.Fatal("after invalidation the removed permission must be denied")
	}
}

func TestMatchPermissionWildcards(t *testing.T) {
	cases := []struct {
		stored, method, route string
		want                  bool
	}{
		{"* *", "GET", "/api/v1/x", true},
		{"*", "DELETE", "/api/v1/anything", true},
		{"GET /api/v1/orders", "GET", "/api/v1/orders", true},
		{"GET /api/v1/orders", "POST", "/api/v1/orders", false},
		{"GET /api/v1/orders", "GET", "/api/v1/orders/9", false},
		{"POST /api/v1/orders/*", "POST", "/api/v1/orders/9", true},
		{"POST /api/v1/orders/*", "POST", "/api/v1/orders/9/items", true},
		{"POST /api/v1/orders/*", "GET", "/api/v1/orders/9", false},
	}
	for _, c := range cases {
		routeKey := c.method + " " + c.route
		if got := matchPermission(c.stored, c.method, routeKey); got != c.want {
			t.Fatalf("matchPermission(%q, %q) = %v, want %v", c.stored, routeKey, got, c.want)
		}
	}
}

func TestCacheTTLApplied(t *testing.T) {
	db, svc, mr := newRbacEnv(t)
	uid := uuid.New()
	seedRBAC(t, db, uid, "GET /api/v1/ping")

	ctx := context.Background()
	svc.Check(ctx, uid, "GET", "/api/v1/ping")

	keys := mr.Keys()
	found := false
	for _, k := range keys {
		if len(k) > 9 && k[:9] == "laundry-p" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no prefixed cache keys written, got %v", keys)
	}
	if got := mr.TTL("laundry-pro:rbac:version"); got != 0 && got != -1 {
		t.Fatalf("version key TTL = %s, want no expiry", got)
	}
	if ttl := mr.TTL("laundry-pro:rbac:user:" + uid.String() + ":v0"); ttl > rbacCacheTTL || ttl <= 0 {
		t.Fatalf("user cache TTL = %s, want ~%s", ttl, rbacCacheTTL)
	}
}
