package settings

import (
	"context"
	"errors"
	"strings"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	apperr "github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service manages the singleton shop configuration.
type Service struct {
	db    *gorm.DB
	audit *audit.Writer
}

// NewService builds the settings service.
func NewService(db *gorm.DB, audit *audit.Writer) *Service {
	return &Service{db: db, audit: audit}
}

// Get returns the shop configuration, creating the default row on first
// access so the endpoint always has something to return.
func (s *Service) Get(ctx context.Context) (*View, error) {
	row, err := s.ensure(ctx)
	if err != nil {
		return nil, err
	}
	return toView(&row), nil
}

// Defaults returns the shop-wide money defaults, implementing
// shopconfig.Provider so money-flow modules share one source of truth.
func (s *Service) Defaults(ctx context.Context) (shopconfig.Defaults, error) {
	row, err := s.ensure(ctx)
	if err != nil {
		return shopconfig.Defaults{}, err
	}
	return shopconfig.Defaults{
		BusinessName:  row.BusinessName,
		Currency:      row.Currency,
		TaxRate:       row.TaxRate,
		InvoicePrefix: row.InvoicePrefix,
		ReceiptFooter: row.ReceiptFooter,
	}, nil
}

// Update mutates the singleton shop configuration, recording an audit entry
// with before/after snapshots (no sensitive data involved).
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	row, err := s.ensure(ctx)
	if err != nil {
		return nil, err
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.BusinessName != nil {
		name := strings.TrimSpace(*req.BusinessName)
		if name == "" {
			return nil, apperr.New(400, "business name cannot be empty")
		}
		updates["business_name"] = name
	}
	if req.BusinessPhone != nil {
		updates["business_phone"] = strings.TrimSpace(*req.BusinessPhone)
	}
	if req.BusinessEmail != nil {
		email := strings.TrimSpace(*req.BusinessEmail)
		if email != "" && !strings.Contains(email, "@") {
			return nil, apperr.New(400, "business email is invalid")
		}
		updates["business_email"] = email
	}
	if req.Address != nil {
		updates["address"] = strings.TrimSpace(*req.Address)
	}
	if req.City != nil {
		updates["city"] = strings.TrimSpace(*req.City)
	}
	if req.Currency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*req.Currency))
		if len(currency) != 3 {
			return nil, apperr.New(400, "currency must be a 3-letter ISO code")
		}
		updates["currency"] = currency
	}
	if req.TaxRate != nil {
		if *req.TaxRate < 0 || *req.TaxRate > 100 {
			return nil, apperr.New(400, "tax rate must be between 0 and 100")
		}
		updates["tax_rate"] = *req.TaxRate
	}
	if req.TaxID != nil {
		updates["tax_id"] = strings.TrimSpace(*req.TaxID)
	}
	if req.InvoicePrefix != nil {
		prefix := strings.ToUpper(strings.TrimSpace(*req.InvoicePrefix))
		if prefix == "" {
			return nil, apperr.New(400, "invoice prefix cannot be empty")
		}
		updates["invoice_prefix"] = prefix
	}
	if req.ReceiptFooter != nil {
		updates["receipt_footer"] = strings.TrimSpace(*req.ReceiptFooter)
	}
	if req.Timezone != nil {
		tz := strings.TrimSpace(*req.Timezone)
		if tz == "" {
			return nil, apperr.New(400, "timezone cannot be empty")
		}
		updates["timezone"] = tz
	}
	if len(updates) == 0 {
		return toView(&row), nil
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.ShopSetting
	if err := s.db.WithContext(ctx).First(&after, "id = ?", row.ID).Error; err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "shop_setting", row.ID.String(), "update", before, &after)
	return toView(&after), nil
}

// singletonID pins the settings row. Because it is the primary key, two
// concurrent first-accesses cannot create duplicate rows: one insert wins
// and the loser re-reads it, so the singleton invariant holds on the hot
// path (every money-flow create resolves the shop defaults).
var singletonID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// ensure returns the singleton row, creating a default one when missing.
func (s *Service) ensure(ctx context.Context) (models.ShopSetting, error) {
	var row models.ShopSetting
	err := s.db.WithContext(ctx).First(&row).Error
	if err == nil {
		return row, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return row, err
	}
	row = defaultSetting()
	row.ID = singletonID
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		// Lost the create race: the winner is the singleton row.
		if err := s.db.WithContext(ctx).First(&row).Error; err != nil {
			return row, err
		}
	}
	return row, nil
}

// defaultSetting is the initial shop configuration.
func defaultSetting() models.ShopSetting {
	return models.ShopSetting{
		BusinessName:  "My Laundry",
		Currency:      "INR",
		TaxRate:       0,
		InvoicePrefix: "INV",
		Timezone:      "Asia/Kolkata",
	}
}

func toView(row *models.ShopSetting) *View {
	return &View{
		ID:            row.ID.String(),
		BusinessName:  row.BusinessName,
		BusinessPhone: row.BusinessPhone,
		BusinessEmail: row.BusinessEmail,
		Address:       row.Address,
		City:          row.City,
		Currency:      row.Currency,
		TaxRate:       row.TaxRate,
		TaxID:         row.TaxID,
		InvoicePrefix: row.InvoicePrefix,
		ReceiptFooter: row.ReceiptFooter,
		Timezone:      row.Timezone,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
