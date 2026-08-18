package garment

import (
	"context"
	"strings"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service manages garments within orders.
type Service struct {
	db    *gorm.DB
	audit *audit.Writer
}

// NewService builds the garment service.
func NewService(db *gorm.DB, audit *audit.Writer) *Service {
	return &Service{db: db, audit: audit}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"order_id": pagination.TypeUUID,
		"name":     pagination.TypeString,
		"type":     pagination.TypeString,
		"status":   pagination.TypeString,
		"quantity": pagination.TypeNumber,
	},
	map[string]pagination.ColumnType{
		"order_id": pagination.TypeUUID,
		"name":     pagination.TypeString,
		"type":     pagination.TypeString,
		"status":   pagination.TypeString,
		"quantity": pagination.TypeNumber,
	},
	"name", "type",
)

// Create adds a garment.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if req.OrderID == uuid.Nil || strings.TrimSpace(req.Name) == "" {
		return nil, errors.New(400, "order_id and name are required")
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	row := models.Garment{
		OrderID:      req.OrderID,
		Name:         strings.TrimSpace(req.Name),
		Type:         req.Type,
		Color:        req.Color,
		Fabric:       req.Fabric,
		Quantity:     req.Quantity,
		Status:       models.GarmentReceived,
		Instructions: req.Instructions,
		DamageNotes:  req.DamageNotes,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "garment", row.ID.String(), "create", nil, &row)
	return toView(&row), nil
}

// Update mutates garment fields including status.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.Garment
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	if req.Fabric != nil {
		updates["fabric"] = *req.Fabric
	}
	if req.Quantity != nil {
		updates["quantity"] = *req.Quantity
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Instructions != nil {
		updates["instructions"] = *req.Instructions
	}
	if req.DamageNotes != nil {
		updates["damage_notes"] = *req.DamageNotes
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.Garment
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "garment", row.ID.String(), "update", before, &after)
	return toView(&after), nil
}

// Delete removes a garment.
func (s *Service) Delete(ctx context.Context, req IDRequest, meta audit.Meta) error {
	var row models.Garment
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "garment", row.ID.String(), "delete", before, nil)
	return nil
}

// List returns cursor-paged garments with summary.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.Garment{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Garment{}), req.PageRequest, spec, &items)
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
	base := s.db.WithContext(ctx).Model(&models.Garment{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		TotalItems int64 `gorm:"column:total_items"`
		Received   int64 `gorm:"column:received"`
		Cleaned    int64 `gorm:"column:cleaned"`
		Ironed     int64 `gorm:"column:ironed"`
		Ready      int64 `gorm:"column:ready"`
		Damaged    int64 `gorm:"column:damaged"`
	}
	if err := filtered.Select(`
		COALESCE(SUM(quantity), 0) AS total_items,
		SUM(CASE WHEN status = 'received' THEN quantity ELSE 0 END) AS received,
		SUM(CASE WHEN status = 'cleaned' THEN quantity ELSE 0 END) AS cleaned,
		SUM(CASE WHEN status = 'ironed' THEN quantity ELSE 0 END) AS ironed,
		SUM(CASE WHEN status = 'ready' THEN quantity ELSE 0 END) AS ready,
		SUM(CASE WHEN damage_notes <> '' THEN 1 ELSE 0 END) AS damaged`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	byStatus := map[string]int64{}
	type st struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:cnt"`
	}
	var rows []st
	s.db.WithContext(ctx).Model(&models.Garment{}).Select("status, COUNT(*) AS cnt").Group("status").Scan(&rows)
	for _, r := range rows {
		byStatus[r.Status] = r.Count
	}
	return &Summary{
		TotalItems: sums.TotalItems, Received: sums.Received, Cleaned: sums.Cleaned,
		Ironed: sums.Ironed, Ready: sums.Ready, Damaged: sums.Damaged, ByStatus: byStatus,
	}, nil
}

func toView(row *models.Garment) *View {
	return &View{
		ID:           row.ID.String(),
		OrderID:      row.OrderID.String(),
		Name:         row.Name,
		Type:         row.Type,
		Color:        row.Color,
		Fabric:       row.Fabric,
		Quantity:     row.Quantity,
		Status:       row.Status,
		Instructions: row.Instructions,
		DamageNotes:  row.DamageNotes,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
