package combo

import (
	"context"
	"fmt"
	"math"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/google/uuid"
)

// attachItems loads combo items in one query and builds views (no N+1).
func (s *Service) attachItems(ctx context.Context, rows []models.Combo) ([]View, error) {
	ids := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].ID)
	}
	byCombo, err := s.itemsByCombo(ctx, ids)
	if err != nil {
		return nil, err
	}
	views := make([]View, 0, len(rows))
	for i := range rows {
		row := rows[i]
		row.Items = byCombo[row.ID]
		views = append(views, *toView(&row))
	}
	return views, nil
}

// itemsByCombo groups combo items by their combo id.
func (s *Service) itemsByCombo(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]models.ComboItem, error) {
	out := map[uuid.UUID][]models.ComboItem{}
	if len(ids) == 0 {
		return out, nil
	}
	items := []models.ComboItem{}
	if err := s.db.WithContext(ctx).Preload("ServiceType").Where("combo_id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, it := range items {
		out[it.ComboID] = append(out[it.ComboID], it)
	}
	return out, nil
}

// summary returns aggregate data for the list endpoint.
func (s *Service) summary(ctx context.Context, req pagination.PageRequest) (*Summary, error) {
	base := s.db.WithContext(ctx).Model(&models.Combo{})
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
	s.db.WithContext(ctx).Model(&models.Combo{}).Select("unit, COUNT(*) AS cnt").Group("unit").Scan(&units)
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

// load fetches a combo with its items and the item service snapshots.
func (s *Service) load(ctx context.Context, id uuid.UUID) (*models.Combo, error) {
	var row models.Combo
	if err := s.db.WithContext(ctx).Preload("Items.ServiceType").First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func toView(row *models.Combo) *View {
	v := &View{
		ID:           row.ID.String(),
		Name:         row.Name,
		Code:         row.Code,
		Description:  row.Description,
		Unit:         row.Unit,
		Price:        row.Price,
		Currency:     row.Currency,
		IsActive:     row.IsActive,
		IsFeatured:   row.IsFeatured,
		ValidityDays: row.ValidityDays,
		RenewalType:  row.RenewalType,
		ValidityText: computeValidityText(row.ValidityDays, row.RenewalType),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
	v.Items = make([]ItemView, 0, len(row.Items))
	var regular float64
	for _, it := range row.Items {
		item := ItemView{ServiceTypeID: it.ServiceTypeID.String(), Quantity: it.Quantity}
		if st := it.ServiceType; st != nil {
			item.ServiceName = st.Name
			item.Unit = st.Unit
			item.Price = st.Price
			item.Currency = st.Currency
			item.Icon = st.Icon
			regular += float64(it.Quantity) * st.Price
		}
		v.Items = append(v.Items, item)
	}
	v.RegularPrice = round2(regular)
	savings := regular - row.Price
	if savings < 0 {
		savings = 0
	}
	v.Savings = round2(savings)
	return v
}

// computeValidityText returns a human-readable validity/renewal string.
func computeValidityText(days int, renewal string) string {
	if days <= 0 {
		switch renewal {
		case "auto":
			return "Renews automatically"
		case "manual":
			return "Renew anytime"
		default:
			return "One-time purchase"
		}
	}
	var period string
	switch {
	case days == 1:
		period = "1 day"
	case days < 7:
		period = fmt.Sprintf("%d days", days)
	case days == 7:
		period = "1 week"
	case days < 30:
		weeks := days / 7
		if days%7 == 0 {
			period = fmt.Sprintf("%d weeks", weeks)
		} else {
			period = fmt.Sprintf("%d days", days)
		}
	case days == 30:
		period = "1 month"
	case days < 365:
		months := days / 30
		if days%30 == 0 {
			period = fmt.Sprintf("%d months", months)
		} else {
			period = fmt.Sprintf("%d days", days)
		}
	case days == 365:
		period = "1 year"
	default:
		years := days / 365
		if days%365 == 0 {
			period = fmt.Sprintf("%d years", years)
		} else {
			period = fmt.Sprintf("%d days", days)
		}
	}
	switch renewal {
	case "auto":
		return fmt.Sprintf("Valid for %s · Renews automatically", period)
	case "manual":
		return fmt.Sprintf("Valid for %s · Renew anytime", period)
	default:
		return fmt.Sprintf("Valid for %s", period)
	}
}

// round2 rounds a money value to 2 decimal places to avoid float drift.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func pageRes(res *pagination.PageResult) *response.Pagination {
	if res == nil {
		return nil
	}
	return &response.Pagination{Limit: res.Limit, Total: res.Total, HasMore: res.HasMore, Offset: res.Offset}
}
