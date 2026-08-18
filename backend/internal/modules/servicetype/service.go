package servicetype

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

// Service manages service types.
type Service struct {
	db      *gorm.DB
	audit   *audit.Writer
	shop    shopconfig.Provider
	catalog catalog.Invalidator // optional; clears the public pricing catalog cache
}

// NewService builds the service type service.
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
		"name":           pagination.TypeString,
		"code":           pagination.TypeString,
		"price":          pagination.TypeNumber,
		"tax_rate":       pagination.TypeNumber,
		"estimated_days": pagination.TypeNumber,
		"is_active":      pagination.TypeBool,
	},
	map[string]pagination.ColumnType{
		"name":           pagination.TypeString,
		"code":           pagination.TypeString,
		"unit":           pagination.TypeString,
		"price":          pagination.TypeNumber,
		"tax_rate":       pagination.TypeNumber,
		"estimated_days": pagination.TypeNumber,
		"is_active":      pagination.TypeBool,
	},
	"name", "code",
)

// Create adds a service type.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Code) == "" {
		return nil, errors.New(400, "name and code are required")
	}
	if req.Price < 0 {
		return nil, errors.New(400, "price cannot be negative")
	}
	defaults, err := s.shop.Defaults(ctx)
	if err != nil {
		return nil, err
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	row := models.ServiceType{
		Name:          strings.TrimSpace(req.Name),
		Code:          strings.ToUpper(strings.TrimSpace(req.Code)),
		Description:   req.Description,
		Unit:          orDefault(req.Unit, "kg"),
		Price:         req.Price,
		Currency:      orDefault(req.Currency, defaults.Currency),
		TaxRate:       req.TaxRate,
		EstimatedDays: req.EstimatedDays,
		IsActive:      active,
		Icon:          req.Icon,
	}
	if row.EstimatedDays == 0 {
		row.EstimatedDays = 2
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUnique(err) {
			return nil, errors.ErrConflict
		}
		return nil, err
	}
	s.audit.Log(ctx, meta, "service_type", row.ID.String(), "create", nil, &row)
	s.invalidateCatalog(ctx)
	return toView(&row), nil
}

// Update mutates a service type.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.ServiceType
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
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
	if req.TaxRate != nil {
		updates["tax_rate"] = *req.TaxRate
	}
	if req.EstimatedDays != nil {
		updates["estimated_days"] = *req.EstimatedDays
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Icon != nil {
		updates["icon"] = *req.Icon
	}
	if len(updates) == 0 {
		return toView(&row), nil
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.ServiceType
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "service_type", row.ID.String(), "update", before, &after)
	s.invalidateCatalog(ctx)
	return toView(&after), nil
}

// ListActive returns sellable service types for the public pricing catalog.
func (s *Service) ListActive(ctx context.Context) ([]View, error) {
	rows := []models.ServiceType{}
	if err := s.db.WithContext(ctx).Where("is_active = ?", true).Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	views := make([]View, 0, len(rows))
	for i := range rows {
		views = append(views, *toView(&rows[i]))
	}
	return views, nil
}

// Delete removes a service type.
func (s *Service) Delete(ctx context.Context, req DeleteRequest, meta audit.Meta) error {
	var row models.ServiceType
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "service_type", row.ID.String(), "delete", before, nil)
	s.invalidateCatalog(ctx)
	return nil
}

// List returns a cursor-paged list with summary data.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.ServiceType{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.ServiceType{}), req.PageRequest, spec, &items)
	if err != nil {
		return nil, nil, nil, err
	}
	views := make([]View, 0, len(items))
	for i := range items {
		views = append(views, *toView(&items[i]))
	}
	summary, err := s.summary(ctx, req.PageRequest)
	if err != nil {
		return nil, nil, nil, err
	}
	return views, pageRes(res), summary, nil
}

func (s *Service) summary(ctx context.Context, req pagination.PageRequest) (*Summary, error) {
	base := s.db.WithContext(ctx).Model(&models.ServiceType{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		TotalActive   int64   `gorm:"column:total_active"`
		TotalInactive int64   `gorm:"column:total_inactive"`
		AvgPrice      float64 `gorm:"column:avg_price"`
		MaxPrice      float64 `gorm:"column:max_price"`
		MinPrice      float64 `gorm:"column:min_price"`
	}
	if err := filtered.Select(`
		SUM(CASE WHEN is_active THEN 1 ELSE 0 END) AS total_active,
		SUM(CASE WHEN NOT is_active THEN 1 ELSE 0 END) AS total_inactive,
		COALESCE(AVG(price), 0) AS avg_price,
		COALESCE(MAX(price), 0) AS max_price,
		COALESCE(MIN(price), 0) AS min_price`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	byUnit := map[string]int64{}
	type unitRow struct {
		Unit  string `gorm:"column:unit"`
		Count int64  `gorm:"column:cnt"`
	}
	var units []unitRow
	s.db.WithContext(ctx).Model(&models.ServiceType{}).Select("unit, COUNT(*) AS cnt").Group("unit").Scan(&units)
	for _, u := range units {
		byUnit[u.Unit] = u.Count
	}
	return &Summary{
		TotalActive:   sums.TotalActive,
		TotalInactive: sums.TotalInactive,
		AvgPrice:      sums.AvgPrice,
		MaxPrice:      sums.MaxPrice,
		MinPrice:      sums.MinPrice,
		ByUnit:        byUnit,
	}, nil
}

func toView(row *models.ServiceType) *View {
	return &View{
		ID:            row.ID.String(),
		Name:          row.Name,
		Code:          row.Code,
		Description:   row.Description,
		Unit:          row.Unit,
		Price:         row.Price,
		Currency:      row.Currency,
		TaxRate:       row.TaxRate,
		EstimatedDays: row.EstimatedDays,
		IsActive:      row.IsActive,
		Icon:          row.Icon,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func pageRes(res *pagination.PageResult) *response.Pagination {
	if res == nil {
		return nil
	}
	return &response.Pagination{Limit: res.Limit, Total: res.Total, HasMore: res.HasMore, Offset: res.Offset}
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func isUnique(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "duplicate key value")
}
