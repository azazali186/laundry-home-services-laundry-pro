package auth

import (
	"strings"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/google/uuid"
)

// roleViews maps roles to public views.
func roleViews(roles []models.Role) []RoleView {
	out := make([]RoleView, 0, len(roles))
	for _, r := range roles {
		out = append(out, RoleView{
			ID:          r.ID.String(),
			Name:        r.Name,
			Code:        r.Code,
			Description: r.Description,
			IsSystem:    r.IsSystem,
			Permissions: permissionViews(r.Permissions),
		})
	}
	return out
}

// permissionViews maps permissions to public views.
func permissionViews(perms []models.Permission) []PermissionView {
	out := make([]PermissionView, 0, len(perms))
	for _, p := range perms {
		out = append(out, PermissionView{
			ID:      p.ID.String(),
			Name:    p.Name,
			Route:   p.Route,
			Path:    p.Path,
			Method:  p.Method,
			Service: p.Service,
			Guard:   p.Guard,
		})
	}
	return out
}

// uuidNil returns the nil UUID for id-required checks.
func uuidNil() uuid.UUID { return uuid.Nil }

// pageResponse converts a pagination result to the response envelope.
func pageResponse(res *pagination.PageResult) *response.Pagination {
	if res == nil {
		return nil
	}
	return &response.Pagination{
		Limit:   res.Limit,
		Total:   res.Total,
		HasMore: res.HasMore,
		Offset:  res.Offset,
	}
}

// isUniqueViolation reports whether the error is a DB unique-constraint
// violation (portable across PostgreSQL and SQLite without raw SQL).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value") ||
		strings.Contains(msg, "SQLSTATE 23505")
}
