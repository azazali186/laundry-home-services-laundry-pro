package payment

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/breaker"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/crypto"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/gateway"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service manages payments with encrypted sensitive fields, gateway calls
// behind a circuit breaker and idempotent capture via distributed locking.
type Service struct {
	db      *gorm.DB
	cache   *cache.Cache
	cipher  *crypto.Cipher
	gateway gateway.Client
	breaker *breaker.Breaker
	bus     *bus.Bus
	audit   *audit.Writer
	shop    shopconfig.Provider
}

// NewService builds the payment service.
func NewService(db *gorm.DB, cache *cache.Cache, cipher *crypto.Cipher, gateway gateway.Client, audit *audit.Writer, bus *bus.Bus, shop shopconfig.Provider) *Service {
	shop = shopconfig.Guard(shop)
	return &Service{
		db: db, cache: cache, cipher: cipher,
		gateway: gateway, breaker: breaker.New("payment-gateway"),
		bus: bus, audit: audit, shop: shop,
	}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"payment_no": pagination.TypeString,
		"order_id":   pagination.TypeUUID,
		"status":     pagination.TypeString,
		"method":     pagination.TypeString,
		"gateway":    pagination.TypeString,
		"amount":     pagination.TypeNumber,
	},
	map[string]pagination.ColumnType{
		"payment_no": pagination.TypeString,
		"order_id":   pagination.TypeUUID,
		"status":     pagination.TypeString,
		"method":     pagination.TypeString,
		"gateway":    pagination.TypeString,
		"amount":     pagination.TypeNumber,
	},
	"payment_no",
)

// Create adds a pending payment, encrypting sensitive fields.
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if req.OrderID == uuid.Nil || req.Amount <= 0 {
		return nil, errors.New(400, "order_id and a positive amount are required")
	}
	defaults, err := s.shop.Defaults(ctx)
	if err != nil {
		return nil, err
	}
	txID, _ := s.cipher.Encrypt(req.TransactionID)
	last4, _ := s.cipher.Encrypt(req.CardLast4)
	row := models.Payment{
		OrderID:       req.OrderID,
		PaymentNo:     modkit.NextNo("PAY"),
		Amount:        req.Amount,
		Currency:      modkit.OrDefault(req.Currency, defaults.Currency),
		Method:        modkit.OrDefault(req.Method, "cash"),
		Status:        models.PaymentPending,
		Gateway:       req.Gateway,
		TransactionID: txID,
		CardLast4:     last4,
		Note:          req.Note,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "payment", row.ID.String(), "create", nil, &row)
	return toView(&row, s.cipher), nil
}

// Update mutates editable payment fields.
func (s *Service) Update(ctx context.Context, req UpdateRequest, meta audit.Meta) (*View, error) {
	var row models.Payment
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	if row.Status == models.PaymentPaid || row.Status == models.PaymentRefunded {
		return nil, errors.New(409, "settled payments cannot be edited")
	}
	before := audit.DiffBefore(&row)
	updates := map[string]any{}
	if req.Method != nil {
		updates["method"] = *req.Method
	}
	if req.Gateway != nil {
		updates["gateway"] = *req.Gateway
	}
	if req.Note != nil {
		updates["note"] = *req.Note
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}
	var after models.Payment
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	s.audit.Log(ctx, meta, "payment", row.ID.String(), "update", before, &after)
	return toView(&after, s.cipher), nil
}

// Delete removes a payment.
func (s *Service) Delete(ctx context.Context, req IDRequest, meta audit.Meta) error {
	var row models.Payment
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return errors.ErrNotFound
	}
	before := audit.DiffBefore(&row)
	if err := s.db.WithContext(ctx).Delete(&row).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "payment", row.ID.String(), "delete", before, nil)
	return nil
}

// List returns offset-paged payments with money summary.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.Payment{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Payment{}), req.PageRequest, spec, &items)
	if err != nil {
		return nil, nil, nil, err
	}
	views := make([]View, 0, len(items))
	for i := range items {
		views = append(views, *toView(&items[i], s.cipher))
	}
	summary, err := s.summary(ctx, req.PageRequest)
	if err != nil {
		return nil, nil, nil, err
	}
	return views, modkit.PageRes(res), summary, nil
}

func (s *Service) summary(ctx context.Context, req pagination.PageRequest) (*Summary, error) {
	base := s.db.WithContext(ctx).Model(&models.Payment{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		Total       int64   `gorm:"column:total"`
		Pending     int64   `gorm:"column:pending"`
		Paid        int64   `gorm:"column:paid"`
		Failed      int64   `gorm:"column:failed"`
		Refunded    int64   `gorm:"column:refunded"`
		Collected   float64 `gorm:"column:collected"`
		Outstanding float64 `gorm:"column:outstanding"`
		RefundedSum float64 `gorm:"column:refunded_sum"`
	}
	if err := filtered.Select(`
		COUNT(*) AS total,
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending,
		SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) AS paid,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed,
		SUM(CASE WHEN status = 'refunded' THEN 1 ELSE 0 END) AS refunded,
		COALESCE(SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END), 0) AS collected,
		COALESCE(SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END), 0) AS outstanding,
		COALESCE(SUM(CASE WHEN status = 'refunded' THEN amount ELSE 0 END), 0) AS refunded_sum`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	byMethod := map[string]int64{}
	type m struct {
		Method string `gorm:"column:method"`
		Count  int64  `gorm:"column:cnt"`
	}
	var rows []m
	s.db.WithContext(ctx).Model(&models.Payment{}).Select("method, COUNT(*) AS cnt").Group("method").Scan(&rows)
	for _, r := range rows {
		byMethod[r.Method] = r.Count
	}
	return &Summary{
		Total: sums.Total, Pending: sums.Pending, Paid: sums.Paid,
		Failed: sums.Failed, Refunded: sums.Refunded,
		Collected: sums.Collected, Outstanding: sums.Outstanding,
		ByMethod: byMethod, RefundedSum: sums.RefundedSum,
	}, nil
}

func toView(row *models.Payment, cipher *crypto.Cipher) *View {
	last4, _ := cipher.Decrypt(row.CardLast4)
	if last4 != "" {
		last4 = maskLast4(last4)
	}
	return &View{
		ID:        row.ID.String(),
		OrderID:   row.OrderID.String(),
		PaymentNo: row.PaymentNo,
		Amount:    row.Amount,
		Currency:  row.Currency,
		Method:    row.Method,
		Status:    row.Status,
		Gateway:   row.Gateway,
		CardLast4: last4,
		PaidAt:    row.PaidAt,
		Note:      row.Note,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func maskLast4(last4 string) string {
	if len(last4) > 4 {
		last4 = last4[len(last4)-4:]
	}
	return "************" + last4
}
