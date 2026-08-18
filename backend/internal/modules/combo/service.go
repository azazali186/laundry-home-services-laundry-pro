// Package combo manages priced bundles of service types (weekly passes,
// family plans). The bundle price is explicit, so discounts are first-class
// and always sourced from the backend.
package combo

import (
	"context"
	"strings"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/catalog"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"gorm.io/gorm"
)

// Service manages combos.
type Service struct {
	db      *gorm.DB
	audit   *audit.Writer
	shop    shopconfig.Provider
	catalog catalog.Invalidator // optional; clears the public pricing catalog cache
}

// NewService builds the combo service.
func NewService(db *gorm.DB, audit *audit.Writer, shop shopconfig.Provider) *Service {
	shop = shopconfig.Guard(shop)
	return &Service{db: db, audit: audit, shop: shop}
}

// SetCatalogInvalidator wires the pricing catalog cache invalidator. Called
// by the container after the pricing service exists (which needs this
// service as a reader, hence the setter rather than a constructor arg).
func (s *Service) SetCatalogInvalidator(inv catalog.Invalidator) { s.catalog = inv }

// invalidateCatalog drops the cached public catalog after a mutation so the
// landing page reflects the change immediately. Best-effort.
func (s *Service) invalidateCatalog(ctx context.Context) {
	if s.catalog != nil {
		s.catalog.InvalidateCatalog(ctx)
	}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"name":      pagination.TypeString,
		"code":      pagination.TypeString,
		"price":     pagination.TypeNumber,
		"is_active": pagination.TypeBool,
	},
	map[string]pagination.ColumnType{
		"name":      pagination.TypeString,
		"code":      pagination.TypeString,
		"unit":      pagination.TypeString,
		"currency":  pagination.TypeString,
		"price":     pagination.TypeNumber,
		"is_active": pagination.TypeBool,
	},
	"name", "code",
)

// Create adds a combo with its items inside a transaction.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	name := strings.TrimSpace(req.Name)
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if name == "" || code == "" {
		return nil, errors.New(400, "name and code are required")
	}
	if req.Price < 0 {
		return nil, errors.New(400, "price cannot be negative")
	}
	if len(req.Items) == 0 {
		return nil, errors.New(400, "at least one item is required")
	}
	if err := s.validateItems(ctx, req.Items); err != nil {
		return nil, err
	}
	defaults, err := s.shop.Defaults(ctx)
	if err != nil {
		return nil, err
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	featured := false
	if req.IsFeatured != nil {
		featured = *req.IsFeatured
	}
	validityDays := 0
	if req.ValidityDays != nil {
		validityDays = *req.ValidityDays
	}
	renewalType := "none"
	if req.RenewalType != nil {
		renewalType = *req.RenewalType
	}
	row := models.Combo{
		Name:         name,
		Code:         code,
		Description:  strings.TrimSpace(req.Description),
		Unit:         orDefault(req.Unit, "per_week"),
		Price:        req.Price,
		Currency:     orDefault(req.Currency, defaults.Currency),
		IsActive:     active,
		IsFeatured:   featured,
		ValidityDays: validityDays,
		RenewalType:  renewalType,
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for _, it := range req.Items {
			if err := tx.Create(&models.ComboItem{ComboID: row.ID, ServiceTypeID: it.ServiceTypeID, Quantity: it.Quantity}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if isUnique(err) {
			return nil, errors.ErrConflict
		}
		return nil, err
	}
	after, err := s.load(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "combo", row.ID.String(), "create", nil, after)
	s.invalidateCatalog(ctx)
	return toView(after), nil
}

// Update mutates a combo, replacing its items when provided.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.Combo
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if req.Price != nil && *req.Price < 0 {
		return nil, errors.New(400, "price cannot be negative")
	}
	if req.Code != nil && strings.TrimSpace(*req.Code) == "" {
		return nil, errors.New(400, "code cannot be empty")
	}
	if req.Items != nil {
		if len(req.Items) == 0 {
			return nil, errors.New(400, "at least one item is required")
		}
		if err := s.validateItems(ctx, req.Items); err != nil {
			return nil, err
		}
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.Code != nil {
		updates["code"] = strings.ToUpper(strings.TrimSpace(*req.Code))
	}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if req.Unit != nil {
		updates["unit"] = *req.Unit
	}
	if req.Price != nil {
		updates["price"] = *req.Price
	}
	if req.Currency != nil {
		updates["currency"] = *req.Currency
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.IsFeatured != nil {
		updates["is_featured"] = *req.IsFeatured
	}
	if req.ValidityDays != nil {
		updates["validity_days"] = *req.ValidityDays
	}
	if req.RenewalType != nil {
		updates["renewal_type"] = *req.RenewalType
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&row).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.Items != nil {
			if err := tx.Where("combo_id = ?", row.ID).Delete(&models.ComboItem{}).Error; err != nil {
				return err
			}
			for _, it := range req.Items {
				if err := tx.Create(&models.ComboItem{ComboID: row.ID, ServiceTypeID: it.ServiceTypeID, Quantity: it.Quantity}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		if isUnique(err) {
			return nil, errors.ErrConflict
		}
		return nil, err
	}
	after, err := s.load(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "combo", row.ID.String(), "update", before, after)
	s.invalidateCatalog(ctx)
	return toView(after), nil
}

// Delete removes a combo (items cascade).
func (s *Service) Delete(ctx context.Context, req DeleteRequest, meta audit.Meta) error {
	var row models.Combo
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "combo", row.ID.String(), "delete", before, nil)
	s.invalidateCatalog(ctx)
	return nil
}

// List returns a cursor-paged list with summary data.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	rows := []models.Combo{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Combo{}), req.PageRequest, spec, &rows)
	if err != nil {
		return nil, nil, nil, err
	}
	views, err := s.attachItems(ctx, rows)
	if err != nil {
		return nil, nil, nil, err
	}
	summary, err := s.summary(ctx, req.PageRequest)
	if err != nil {
		return nil, nil, nil, err
	}
	return views, pageRes(res), summary, nil
}

// ListActive returns sellable combos with items for the public catalog.
func (s *Service) ListActive(ctx context.Context) ([]View, error) {
	rows := []models.Combo{}
	if err := s.db.WithContext(ctx).Where("is_active = ?", true).Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return s.attachItems(ctx, rows)
}
