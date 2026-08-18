package auth

import (
	"encoding/json"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

// ---------- auth ----------

// LoginRequest body: POST /api/v1/auth/login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse returned after successful authentication.
type LoginResponse struct {
	AccessToken string   `json:"access_token"`
	ExpiresIn   int64    `json:"expires_in"`
	TokenType   string   `json:"token_type"`
	User        UserView `json:"user"`
}

// ChangePasswordRequest body.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ---------- users ----------

// UserView is the public representation of a user.
type UserView struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Roles       []RoleView `json:"roles,omitempty"`
}

// CreateUserRequest body.
type CreateUserRequest struct {
	Name     string      `json:"name"`
	Email    string      `json:"email"`
	Phone    string      `json:"phone"`
	Password string      `json:"password"`
	RoleIDs  []uuid.UUID `json:"role_ids"`
}

// UpdateUserRequest body (id in body; no path variables).
type UpdateUserRequest struct {
	ID       uuid.UUID   `json:"id"`
	Name     *string     `json:"name"`
	Phone    *string     `json:"phone"`
	IsActive *bool       `json:"is_active"`
	Password *string     `json:"password"`
	RoleIDs  []uuid.UUID `json:"role_ids"`
}

// DeleteUserRequest body.
type DeleteUserRequest struct {
	ID uuid.UUID `json:"id"`
}

// ListUsersRequest body for offset pagination.
type ListUsersRequest struct {
	pagination.PageRequest
}

// UserSummary returned alongside user lists.
type UserSummary struct {
	TotalActive   int64 `json:"total_active"`
	TotalInactive int64 `json:"total_inactive"`
	TotalAdmins   int64 `json:"total_admins"`
	NewThisMonth  int64 `json:"new_this_month"`
}

// ---------- roles ----------

// RoleView is the public representation of a role.
type RoleView struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Code        string           `json:"code"`
	Description string           `json:"description"`
	IsSystem    bool             `json:"is_system"`
	Permissions []PermissionView `json:"permissions,omitempty"`
}

// CreateRoleRequest body.
type CreateRoleRequest struct {
	Name          string      `json:"name"`
	Code          string      `json:"code"`
	Description   string      `json:"description"`
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}

// UpdateRoleRequest body.
type UpdateRoleRequest struct {
	ID            uuid.UUID   `json:"id"`
	Name          *string     `json:"name"`
	Code          *string     `json:"code"`
	Description   *string     `json:"description"`
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}

// DeleteRoleRequest body.
type DeleteRoleRequest struct {
	ID uuid.UUID `json:"id"`
}

// AssignPermissionsRequest body.
type AssignPermissionsRequest struct {
	RoleID        uuid.UUID   `json:"role_id"`
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}

// ListRolesRequest body.
type ListRolesRequest struct {
	pagination.PageRequest
}

// ---------- permissions ----------

// PermissionView is the public representation of a permission.
type PermissionView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Route   string `json:"route"`
	Path    string `json:"path"`
	Method  string `json:"method"`
	Service string `json:"service"`
	Guard   string `json:"guard"`
}

// ListPermissionsRequest body.
type ListPermissionsRequest struct {
	pagination.PageRequest
}

// ListLoginLogsRequest body.
type ListLoginLogsRequest struct {
	pagination.PageRequest
}

// LoginLogView public representation of a login attempt.
type LoginLogView struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// ListAuditLogsRequest body.
type ListAuditLogsRequest struct {
	pagination.PageRequest
}

// AuditLogView public representation of an audit entry.
type AuditLogView struct {
	ID        string          `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	Entity    string          `json:"entity"`
	EntityID  string          `json:"entity_id"`
	Action    string          `json:"action"`
	Method    string          `json:"method"`
	Path      string          `json:"path"`
	IP        string          `json:"ip"`
	Before    json.RawMessage `json:"before"`
	After     json.RawMessage `json:"after"`
	CreatedAt time.Time       `json:"created_at"`
}
