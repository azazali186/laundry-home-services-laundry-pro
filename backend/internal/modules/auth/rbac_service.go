package auth

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// rbacCacheTTL is how long a user's permission set is cached.
const rbacCacheTTL = 5 * time.Minute

// RbacService resolves whether a user may access a route. Permission sets
// are cached in Redis and invalidated on role/permission changes.
type RbacService struct {
	db    *gorm.DB
	cache *cache.Cache
}

// NewRbacService builds the RBAC service.
func NewRbacService(db *gorm.DB, cache *cache.Cache) *RbacService {
	return &RbacService{db: db, cache: cache}
}

// Check reports whether the user may access method+path.
func (r *RbacService) Check(ctx context.Context, userID uuid.UUID, method, path string) bool {
	routes, err := r.permissionRoutes(ctx, userID)
	if err != nil {
		logger.Warn("rbac load failed for %s: %v", userID, err)
		return false
	}
	routeKey := method + " " + path
	for _, allowed := range routes {
		if matchPermission(allowed, method, routeKey) {
			return true
		}
	}
	return false
}

// matchPermission compares a stored permission route against the request.
// Supports "*" method wildcards and trailing "/*" path wildcards.
func matchPermission(stored, method, routeKey string) bool {
	if stored == "*" || stored == "* *" {
		return true
	}
	parts := strings.SplitN(stored, " ", 2)
	if len(parts) != 2 {
		return stored == routeKey
	}
	storedMethod, storedPath := parts[0], parts[1]
	if storedMethod != "*" && storedMethod != method {
		return false
	}
	if strings.HasSuffix(storedPath, "/*") {
		prefix := strings.TrimSuffix(storedPath, "*")
		// The route key carries a "METHOD " prefix; compare against the path
		// portion only so path wildcards match regardless of method.
		reqPath := routeKey
		if i := strings.Index(routeKey, " "); i >= 0 {
			reqPath = routeKey[i+1:]
		}
		return strings.HasPrefix(reqPath, prefix)
	}
	return routeKey == stored
}

// permissionRoutes loads the user's permission routes with Redis caching.
// Cache keys embed a global version so permission changes invalidate every
// user's snapshot without per-key deletes.
func (r *RbacService) permissionRoutes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	version, _ := r.cache.Get(ctx, "rbac:version")
	if version == "" {
		version = "0"
	}
	cacheKey := "rbac:user:" + userID.String() + ":v" + version
	var routes []string
	if err := r.cache.GetJSON(ctx, cacheKey, &routes); err == nil {
		return routes, nil
	}

	var user models.User
	if err := r.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("load user permissions: %w", err)
	}
	if !user.IsActive {
		return nil, nil
	}
	seen := map[string]bool{}
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			if perm.Route == "" || seen[perm.Route] {
				continue
			}
			seen[perm.Route] = true
			routes = append(routes, perm.Route)
		}
	}
	if err := r.cache.SetJSON(ctx, cacheKey, routes, rbacCacheTTL); err != nil {
		logger.Warn("rbac cache write failed: %v", err)
	}
	return routes, nil
}

// Invalidate bumps the global version, invalidating all cached permission sets.
func (r *RbacService) Invalidate(ctx context.Context) {
	version, _ := r.cache.Get(ctx, "rbac:version")
	// Start at "1": the default (absent) version is "0", so the first
	// invalidation must produce a value that differs from it.
	next := "1"
	if v, err := strconv.Atoi(version); err == nil {
		next = strconv.Itoa(v + 1)
	}
	if err := r.cache.Set(ctx, "rbac:version", next, 0); err != nil {
		logger.Warn("rbac version bump failed: %v", err)
	}
}
