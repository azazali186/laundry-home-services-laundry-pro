package auth

import (
	"context"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/crypto"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/password"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserService manages user accounts.
type UserService struct {
	db     *gorm.DB
	cache  *cache.Cache
	cipher *crypto.Cipher
	cfg    *config.Config
	audit  *audit.Writer
	rbac   *RbacService
}

// NewUserService builds the user service.
func NewUserService(db *gorm.DB, cache *cache.Cache, cipher *crypto.Cipher, cfg *config.Config, audit *audit.Writer, rbac *RbacService) *UserService {
	return &UserService{db: db, cache: cache, cipher: cipher, cfg: cfg, audit: audit, rbac: rbac}
}

var userSpec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"name":          pagination.TypeString,
		"email":         pagination.TypeString,
		"is_active":     pagination.TypeBool,
		"last_login_at": pagination.TypeTime,
	},
	map[string]pagination.ColumnType{
		"name":          pagination.TypeString,
		"email":         pagination.TypeString,
		"is_active":     pagination.TypeBool,
		"last_login_at": pagination.TypeTime,
	},
	"name", "email",
)

// Create adds a new user with optional roles.
func (s *UserService) Create(ctx context.Context, req CreateUserRequest, meta audit.Meta) (*UserView, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		return nil, errors.New(400, "name and email are required")
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if len(req.Password) < 8 {
		return nil, errors.New(400, "password must be at least 8 characters")
	}
	var exists int64
	s.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", req.Email).Count(&exists)
	if exists > 0 {
		return nil, errors.ErrEmailInUse
	}
	hashed, err := password.Hash(req.Password, s.cfg.PasswordCost)
	if err != nil {
		return nil, err
	}
	encryptedPhone, _ := s.cipher.Encrypt(req.Phone)

	user := models.User{
		Name:     strings.TrimSpace(req.Name),
		Email:    req.Email,
		Phone:    encryptedPhone,
		Password: hashed,
		IsActive: true,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return s.assignRoles(ctx, tx, user.ID, req.RoleIDs)
	})
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "user", user.ID.String(), "create", nil, &user)
	s.rbac.Invalidate(ctx)
	return s.loadView(ctx, user.ID)
}

// Update applies field-level changes to a user.
func (s *UserService) Update(ctx context.Context, req UpdateUserRequest, meta audit.Meta) (*UserView, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	before := audit.DiffBefore(&user)
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Phone != nil {
		enc, _ := s.cipher.Encrypt(*req.Phone)
		updates["phone"] = enc
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Password != nil && *req.Password != "" {
		hashed, err := password.Hash(*req.Password, s.cfg.PasswordCost)
		if err != nil {
			return nil, err
		}
		updates["password"] = hashed
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&user).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.RoleIDs != nil {
			return s.replaceRoles(ctx, tx, user.ID, req.RoleIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var after models.User
	s.db.WithContext(ctx).Preload("Roles").First(&after, "id = ?", user.ID)
	s.audit.Log(ctx, meta, "user", user.ID.String(), "update", before, &after)
	s.rbac.Invalidate(ctx)
	return s.loadView(ctx, user.ID)
}

// Delete soft-deletes a user and kills their sessions.
func (s *UserService) Delete(ctx context.Context, req DeleteUserRequest, meta audit.Meta) error {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&user)
	if err := s.db.WithContext(ctx).Delete(&user).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "user", user.ID.String(), "delete", before, nil)
	s.rbac.Invalidate(ctx)
	_ = s.cache.Del(ctx, "session:"+user.ID.String())
	return nil
}

// List returns a cursor-paged user list with summary data.
func (s *UserService) List(ctx context.Context, req ListUsersRequest) ([]UserView, *response.Pagination, *UserSummary, error) {
	items := []models.User{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.User{}), req.PageRequest, userSpec, &items)
	if err != nil {
		return nil, nil, nil, err
	}
	views := make([]UserView, 0, len(items))
	for i := range items {
		phone, _ := s.cipher.Decrypt(items[i].Phone)
		views = append(views, UserView{
			ID:          items[i].ID,
			Name:        items[i].Name,
			Email:       items[i].Email,
			Phone:       phone,
			IsActive:    items[i].IsActive,
			LastLoginAt: items[i].LastLoginAt,
			CreatedAt:   items[i].CreatedAt,
			UpdatedAt:   items[i].UpdatedAt,
		})
	}
	summary, err := s.summary(ctx, req.PageRequest)
	if err != nil {
		return nil, nil, nil, err
	}
	return views, pageResponse(res), summary, nil
}

// summary computes aggregate stats for the user list.
func (s *UserService) summary(ctx context.Context, req pagination.PageRequest) (*UserSummary, error) {
	base := s.db.WithContext(ctx).Model(&models.User{})
	filtered, err := pagination.FilteredQuery(base, req, userSpec)
	if err != nil {
		return nil, err
	}
	monthStart := time.Now().AddDate(0, -1, 0)
	var out UserSummary
	if err := filtered.Select(`
		SUM(CASE WHEN is_active THEN 1 ELSE 0 END) AS total_active,
		SUM(CASE WHEN NOT is_active THEN 1 ELSE 0 END) AS total_inactive,
		SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END) AS new_this_month`, monthStart).Scan(&out).Error; err != nil {
		return nil, err
	}
	var admins int64
	s.db.WithContext(ctx).Model(&models.User{}).
		Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Joins("JOIN roles ON roles.id = user_roles.role_id AND roles.code = 'super_admin'").
		Count(&admins)
	out.TotalAdmins = admins
	return &out, nil
}

// loadView loads a user with roles into a public view.
func (s *UserService) loadView(ctx context.Context, userID uuid.UUID) (*UserView, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", userID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	phone, _ := s.cipher.Decrypt(user.Phone)
	return &UserView{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Phone:       phone,
		IsActive:    user.IsActive,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Roles:       roleViews(user.Roles),
	}, nil
}

func (s *UserService) assignRoles(ctx context.Context, tx *gorm.DB, userID uuid.UUID, roleIDs []uuid.UUID) error {
	if len(roleIDs) == 0 {
		return nil
	}
	var roles []models.Role
	if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return err
	}
	return tx.Model(&models.User{Base: models.Base{ID: userID}}).Association("Roles").Replace(roles)
}

func (s *UserService) replaceRoles(ctx context.Context, tx *gorm.DB, userID uuid.UUID, roleIDs []uuid.UUID) error {
	var roles []models.Role
	if len(roleIDs) > 0 {
		if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return err
		}
	}
	return tx.Model(&models.User{Base: models.Base{ID: userID}}).Association("Roles").Replace(roles)
}
