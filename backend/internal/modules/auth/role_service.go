package auth

import (
	"context"
	"strings"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleService manages roles and their permission sets.
type RoleService struct {
	db    *gorm.DB
	audit *audit.Writer
	rbac  *RbacService
}

// NewRoleService builds the role service.
func NewRoleService(db *gorm.DB, audit *audit.Writer, rbac *RbacService) *RoleService {
	return &RoleService{db: db, audit: audit, rbac: rbac}
}

var roleSpec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"name":      pagination.TypeString,
		"code":      pagination.TypeString,
		"is_system": pagination.TypeBool,
	},
	map[string]pagination.ColumnType{
		"name":      pagination.TypeString,
		"code":      pagination.TypeString,
		"is_system": pagination.TypeBool,
	},
	"name", "code",
)

// Create adds a role with optional permissions.
func (s *RoleService) Create(ctx context.Context, req CreateRoleRequest, meta audit.Meta) (*RoleView, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Code) == "" {
		return nil, errors.New(400, "name and code are required")
	}
	role := models.Role{
		Name:        strings.TrimSpace(req.Name),
		Code:        strings.ToLower(strings.TrimSpace(req.Code)),
		Description: req.Description,
		IsSystem:    false,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			if isUniqueViolation(err) {
				return errors.ErrConflict
			}
			return err
		}
		return s.replacePermissions(ctx, tx, role.ID, req.PermissionIDs)
	})
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "role", role.ID.String(), "create", nil, &role)
	s.rbac.Invalidate(ctx)
	return s.loadView(ctx, role.ID)
}

// Update mutates role fields and/or its permission set.
func (s *RoleService) Update(ctx context.Context, req UpdateRoleRequest, meta audit.Meta) (*RoleView, error) {
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if role.IsSystem && (req.Name != nil || req.Code != nil) {
		return nil, errors.New(400, "system roles cannot be renamed")
	}
	before := audit.DiffBefore(&role)
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		updates["code"] = strings.ToLower(strings.TrimSpace(*req.Code))
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&role).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.PermissionIDs != nil {
			return s.replacePermissions(ctx, tx, role.ID, req.PermissionIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var after models.Role
	s.db.WithContext(ctx).Preload("Permissions").First(&after, "id = ?", role.ID)
	s.audit.Log(ctx, meta, "role", role.ID.String(), "update", before, &after)
	s.rbac.Invalidate(ctx)
	return s.loadView(ctx, role.ID)
}

// Delete removes a non-system role.
func (s *RoleService) Delete(ctx context.Context, req DeleteRoleRequest, meta audit.Meta) error {
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	if role.IsSystem {
		return errors.New(400, "system roles cannot be deleted")
	}
	before := audit.DiffBefore(&role)
	if err := s.db.WithContext(ctx).Delete(&role).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "role", role.ID.String(), "delete", before, nil)
	s.rbac.Invalidate(ctx)
	return nil
}

// AssignPermissions replaces a role's permission set.
func (s *RoleService) AssignPermissions(ctx context.Context, req AssignPermissionsRequest, meta audit.Meta) (*RoleView, error) {
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", req.RoleID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	before := audit.DiffBefore(&role)
	if err := s.replacePermissions(ctx, s.db, role.ID, req.PermissionIDs); err != nil {
		return nil, err
	}
	var after models.Role
	s.db.WithContext(ctx).Preload("Permissions").First(&after, "id = ?", role.ID)
	s.audit.Log(ctx, meta, "role", role.ID.String(), "assign_permissions", before, &after)
	s.rbac.Invalidate(ctx)
	return s.loadView(ctx, role.ID)
}

// List returns cursor-paged roles.
func (s *RoleService) List(ctx context.Context, req ListRolesRequest) ([]RoleView, *response.Pagination, error) {
	items := []models.Role{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Role{}), req.PageRequest, roleSpec, &items)
	if err != nil {
		return nil, nil, err
	}
	views := make([]RoleView, 0, len(items))
	for i := range items {
		views = append(views, RoleView{
			ID:          items[i].ID.String(),
			Name:        items[i].Name,
			Code:        items[i].Code,
			Description: items[i].Description,
			IsSystem:    items[i].IsSystem,
		})
	}
	return views, pageResponse(res), nil
}

// ListPermissions returns cursor-paged permissions.
func (s *RoleService) ListPermissions(ctx context.Context, req ListPermissionsRequest) ([]PermissionView, *response.Pagination, error) {
	items := []models.Permission{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Permission{}), req.PageRequest, permissionSpec, &items)
	if err != nil {
		return nil, nil, err
	}
	return permissionViews(items), pageResponse(res), nil
}

var permissionSpec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"name":    pagination.TypeString,
		"route":   pagination.TypeString,
		"path":    pagination.TypeString,
		"method":  pagination.TypeString,
		"service": pagination.TypeString,
	},
	map[string]pagination.ColumnType{
		"name":    pagination.TypeString,
		"route":   pagination.TypeString,
		"path":    pagination.TypeString,
		"method":  pagination.TypeString,
		"service": pagination.TypeString,
	},
	"name", "path",
)

func (s *RoleService) replacePermissions(ctx context.Context, tx *gorm.DB, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	var perms []models.Permission
	if len(permissionIDs) > 0 {
		if err := tx.Where("id IN ?", permissionIDs).Find(&perms).Error; err != nil {
			return err
		}
	}
	return tx.Model(&models.Role{Base: models.Base{ID: roleID}}).Association("Permissions").Replace(perms)
}

func (s *RoleService) loadView(ctx context.Context, roleID uuid.UUID) (*RoleView, error) {
	var role models.Role
	if err := s.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", roleID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	return &RoleView{
		ID:          role.ID.String(),
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		Permissions: permissionViews(role.Permissions),
	}, nil
}
