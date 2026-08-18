package auth

import (
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/router"
)

// Register mounts authenticated routes on the protected /api/v1 group.
// Paths are relative to the group prefix.
func Register(rg router.Registrar, handler *Handler) {
	rg.POST("/auth/logout", handler.Logout)
	rg.POST("/auth/me", handler.Me)
	rg.POST("/auth/change-password", handler.ChangePassword)

	rg.POST("/users", handler.CreateUser)
	rg.PATCH("/users", handler.UpdateUser)
	rg.DELETE("/users", handler.DeleteUser)
	rg.POST("/users/list", handler.ListUsers)

	rg.POST("/roles", handler.CreateRole)
	rg.PATCH("/roles", handler.UpdateRole)
	rg.DELETE("/roles", handler.DeleteRole)
	rg.POST("/roles/assign-permissions", handler.AssignPermissions)
	rg.POST("/roles/list", handler.ListRoles)

	rg.POST("/permissions/list", handler.ListPermissions)

	rg.POST("/login-logs/list", handler.ListLoginLogs)
	rg.POST("/audit-logs/list", handler.ListAuditLogs)
}
