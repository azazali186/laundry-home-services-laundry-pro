package inventory

import (
	"context"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/lock"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"gorm.io/gorm"
)

// Service manages inventory items and stock levels.
type Service struct {
	db    *gorm.DB
	cache *cache.Cache
	audit *audit.Writer
}

// NewService builds the inventory service.
func NewService(db *gorm.DB, cache *cache.Cache, audit *audit.Writer) *Service {
	return &Service{db: db, cache: cache, audit: audit}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"name":             pagination.TypeString,
		"sku":              pagination.TypeString,
		"category":         pagination.TypeString,
		"quantity_on_hand": pagination.TypeNumber,
		"reorder_level":    pagination.TypeNumber,
		"is_active":        pagination.TypeBool,
	},
	map[string]pagination.ColumnType{
		"name":             pagination.TypeString,
		"sku":              pagination.TypeString,
		"category":         pagination.TypeString,
		"quantity_on_hand": pagination.TypeNumber,
		"reorder_level":    pagination.TypeNumber,
		"is_active":        pagination.TypeBool,
	},
	"name", "sku", "category",
)

// Create adds an inventory item.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.SKU) == "" {
		return nil, errors.New(400, "name and sku are required")
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	row := models.InventoryItem{
		Name:           strings.TrimSpace(req.Name),
		SKU:            strings.ToUpper(strings.TrimSpace(req.SKU)),
		Category:       req.Category,
		QuantityOnHand: req.QuantityOnHand,
		ReorderLevel:   req.ReorderLevel,
		Unit:           modkit.OrDefault(req.Unit, "unit"),
		UnitCost:       req.UnitCost,
		IsActive:       active,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		if modkit.IsUnique(err) {
			return nil, errors.ErrConflict
		}
		return nil, err
	}
	s.audit.Log(ctx, meta, "inventory", row.ID.String(), "create", nil, &row)
	return toView(&row), nil
}

// Update mutates item metadata.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.InventoryItem
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.ReorderLevel != nil {
		updates["reorder_level"] = *req.ReorderLevel
	}
	if req.Unit != nil {
		updates["unit"] = *req.Unit
	}
	if req.UnitCost != nil {
		updates["unit_cost"] = *req.UnitCost
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.InventoryItem
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "inventory", row.ID.String(), "update", before, &after)
	return toView(&after), nil
}

// AdjustStock applies a signed delta under a distributed lock, so concurrent
// consumptions/restocks never race on the same item.
func (s *Service) AdjustStock(ctx context.Context, req AdjustRequest, meta audit.Meta) (*View, error) {
	mtx, err := lock.Acquire(ctx, s.cache, "inventory:adjust:"+req.ID.String(), 10*time.Second, 3*time.Second)
	if err != nil {
		return nil, errors.New(409, "stock update in progress")
	}
	defer mtx.Release(ctx)

	var row models.InventoryItem
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	newQty := row.QuantityOnHand + req.Delta
	if newQty < 0 {
		return nil, errors.New(400, "insufficient stock")
	}
	if err := s.db.WithContext(ctx).Model(&row).Update("quantity_on_hand", newQty).Error; err != nil {
		return nil, err
	}
	var after models.InventoryItem
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "inventory", row.ID.String(), "adjust_stock", before, &after)
	return toView(&after), nil
}

// Delete removes an item.
func (s *Service) Delete(ctx context.Context, req IDRequest, meta audit.Meta) error {
	var row models.InventoryItem
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "inventory", row.ID.String(), "delete", before, nil)
	return nil
}

// List returns cursor-paged items with summary.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.InventoryItem{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.InventoryItem{}), req.PageRequest, spec, &items)
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
	return views, modkit.PageRes(res), summary, nil
}

func (s *Service) summary(ctx context.Context, req pagination.PageRequest) (*Summary, error) {
	base := s.db.WithContext(ctx).Model(&models.InventoryItem{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		TotalItems  int64   `gorm:"column:total_items"`
		LowStock    int64   `gorm:"column:low_stock"`
		OutOfStock  int64   `gorm:"column:out_of_stock"`
		TotalValue  float64 `gorm:"column:total_value"`
		ActiveItems int64   `gorm:"column:active_items"`
	}
	if err := filtered.Select(`
		COUNT(*) AS total_items,
		SUM(CASE WHEN quantity_on_hand > 0 AND quantity_on_hand <= reorder_level THEN 1 ELSE 0 END) AS low_stock,
		SUM(CASE WHEN quantity_on_hand <= 0 THEN 1 ELSE 0 END) AS out_of_stock,
		COALESCE(SUM(quantity_on_hand * unit_cost), 0) AS total_value,
		SUM(CASE WHEN is_active THEN 1 ELSE 0 END) AS active_items`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	byCategory := map[string]int64{}
	type cat struct {
		Category string `gorm:"column:category"`
		Count    int64  `gorm:"column:cnt"`
	}
	var cats []cat
	s.db.WithContext(ctx).Model(&models.InventoryItem{}).Select("category, COUNT(*) AS cnt").Group("category").Scan(&cats)
	for _, c := range cats {
		byCategory[c.Category] = c.Count
	}
	return &Summary{
		TotalItems: sums.TotalItems, LowStock: sums.LowStock, OutOfStock: sums.OutOfStock,
		TotalValue: sums.TotalValue, ActiveItems: sums.ActiveItems, ByCategory: byCategory,
	}, nil
}

func toView(row *models.InventoryItem) *View {
	return &View{
		ID:             row.ID.String(),
		Name:           row.Name,
		SKU:            row.SKU,
		Category:       row.Category,
		QuantityOnHand: row.QuantityOnHand,
		ReorderLevel:   row.ReorderLevel,
		Unit:           row.Unit,
		UnitCost:       row.UnitCost,
		IsActive:       row.IsActive,
		LowStock:       row.QuantityOnHand > 0 && row.QuantityOnHand <= row.ReorderLevel,
		StockValue:     row.QuantityOnHand * row.UnitCost,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
