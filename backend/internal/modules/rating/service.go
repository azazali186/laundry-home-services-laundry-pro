package rating

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service manages customer ratings.
type Service struct {
	db    *gorm.DB
	audit *audit.Writer
}

// NewService builds the rating service.
func NewService(db *gorm.DB, audit *audit.Writer) *Service {
	return &Service{db: db, audit: audit}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"order_id":    pagination.TypeUUID,
		"customer_id": pagination.TypeUUID,
		"score":       pagination.TypeNumber,
		"is_public":   pagination.TypeBool,
	},
	map[string]pagination.ColumnType{
		"order_id":    pagination.TypeUUID,
		"customer_id": pagination.TypeUUID,
		"score":       pagination.TypeNumber,
		"is_public":   pagination.TypeBool,
	},
	"comment",
)

// Create adds a rating.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if req.OrderID == uuid.Nil || req.Score < 1 || req.Score > 5 {
		return nil, errors.New(400, "order_id and a score between 1 and 5 are required")
	}
	row := models.Rating{
		OrderID:    req.OrderID,
		CustomerID: req.CustomerID,
		Score:      req.Score,
		Comment:    req.Comment,
		IsPublic:   req.IsPublic,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "rating", row.ID.String(), "create", nil, &row)
	return toView(&row), nil
}

// Update mutates a rating.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.Rating
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.Score != nil {
		if *req.Score < 1 || *req.Score > 5 {
			return nil, errors.New(400, "score must be between 1 and 5")
		}
		updates["score"] = *req.Score
	}
	if req.Comment != nil {
		updates["comment"] = *req.Comment
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.Rating
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "rating", row.ID.String(), "update", before, &after)
	return toView(&after), nil
}

// Delete removes a rating.
func (s *Service) Delete(ctx context.Context, req IDRequest, meta audit.Meta) error {
	var row models.Rating
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "rating", row.ID.String(), "delete", before, nil)
	return nil
}

// List returns cursor-paged ratings with summary.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.Rating{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Rating{}), req.PageRequest, spec, &items)
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
	base := s.db.WithContext(ctx).Model(&models.Rating{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		Total int64   `gorm:"column:total"`
		Avg   float64 `gorm:"column:avg_score"`
		Five  int64   `gorm:"column:five_star"`
		Four  int64   `gorm:"column:four_star"`
		Three int64   `gorm:"column:three_star"`
		Two   int64   `gorm:"column:two_star"`
		One   int64   `gorm:"column:one_star"`
	}
	if err := filtered.Select(`
		COUNT(*) AS total,
		COALESCE(AVG(score), 0) AS avg_score,
		SUM(CASE WHEN score = 5 THEN 1 ELSE 0 END) AS five_star,
		SUM(CASE WHEN score = 4 THEN 1 ELSE 0 END) AS four_star,
		SUM(CASE WHEN score = 3 THEN 1 ELSE 0 END) AS three_star,
		SUM(CASE WHEN score = 2 THEN 1 ELSE 0 END) AS two_star,
		SUM(CASE WHEN score = 1 THEN 1 ELSE 0 END) AS one_star`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	return &Summary{
		Total: sums.Total, AvgScore: sums.Avg, FiveStar: sums.Five,
		FourStar: sums.Four, ThreeStar: sums.Three, TwoStar: sums.Two, OneStar: sums.One,
	}, nil
}

func toView(row *models.Rating) *View {
	return &View{
		ID:         row.ID.String(),
		OrderID:    row.OrderID.String(),
		CustomerID: row.CustomerID.String(),
		Score:      row.Score,
		Comment:    row.Comment,
		IsPublic:   row.IsPublic,
		CreatedAt:  row.CreatedAt,
	}
}
