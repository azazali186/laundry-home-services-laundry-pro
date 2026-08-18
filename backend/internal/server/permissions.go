package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/cloudwego/hertz/pkg/app/server"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

// RouteInfo describes a registered route for permission seeding.
type RouteInfo struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Guard  string `json:"guard"`
	Method string `json:"method"`
}

// IsExcludedRoute reports whether a path must never become a permission
// (login, public pricing, swagger, metrics, health, websocket handshake).
func IsExcludedRoute(path string) bool {
	lower := strings.ToLower(path)
	for _, prefix := range []string{
		"/swagger", "/metrics", "/api/v1/health", "/ws", "/api/v1/pricing",
	} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	if strings.HasSuffix(lower, "/auth/login") || strings.HasSuffix(lower, "/auth/change-password") {
		return true
	}
	return false
}

// SeedPermissions walks the registered routes and upserts a Permission row
// for each auth-protected route (GORM only, no raw SQL).
func SeedPermissions(ctx context.Context, db *gorm.DB, h *server.Hertz) error {
	seen := make(map[string]bool)
	var count int
	for _, r := range h.Routes() {
		routeKey := fmt.Sprintf("%s %s", r.Method, r.Path)
		if seen[routeKey] {
			continue
		}
		seen[routeKey] = true
		if IsExcludedRoute(r.Path) {
			continue
		}
		info := RouteInfo{
			Name:   formatRouteName(r.Path),
			URL:    r.Path,
			Guard:  "API",
			Method: r.Method,
		}
		if err := upsertPermission(ctx, db, info); err != nil {
			return err
		}
		count++
	}
	logger.Info("permission seeding complete: %d routes registered as permissions", count)
	return nil
}

// upsertPermission creates or updates a permission for a route.
func upsertPermission(ctx context.Context, db *gorm.DB, route RouteInfo) error {
	uniqueRoute := fmt.Sprintf("%s %s", route.Method, route.URL)
	var existing models.Permission
	err := db.WithContext(ctx).Where("route = ?", uniqueRoute).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("query permission: %w", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		newPermission := models.Permission{
			Name:    route.Name,
			Route:   uniqueRoute,
			Path:    route.URL,
			Method:  route.Method,
			Service: "api-gateway",
			Guard:   route.Guard,
		}
		if err := db.WithContext(ctx).Create(&newPermission).Error; err != nil {
			return fmt.Errorf("insert permission: %w", err)
		}
		logger.Info("new permission inserted: %s", uniqueRoute)
		return nil
	}
	updates := map[string]any{}
	if existing.Name != route.Name {
		updates["name"] = route.Name
	}
	if existing.Path != route.URL {
		updates["path"] = route.URL
	}
	if existing.Service != "api-gateway" {
		updates["service"] = "api-gateway"
	}
	if existing.Guard != route.Guard {
		updates["guard"] = route.Guard
	}
	if len(updates) > 0 {
		if err := db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return fmt.Errorf("update permission: %w", err)
		}
		logger.Info("permission updated: %s", uniqueRoute)
	}
	return nil
}

// formatRouteName converts a path into a readable permission name,
// mirroring the reference implementation.
func formatRouteName(path string) string {
	cleaned := strings.Replace(path, "/api/v1", "", 1)
	cleaned = strings.ReplaceAll(cleaned, "/", " ")
	titleCaser := cases.Title(language.English)
	return titleCaser.String(strings.TrimSpace(cleaned))
}
