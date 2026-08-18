package notification

import (
	"context"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service manages outbound notifications.
type Service struct {
	db    *gorm.DB
	audit *audit.Writer
}

// NewService builds the notification service.
func NewService(db *gorm.DB, audit *audit.Writer) *Service {
	return &Service{db: db, audit: audit}
}

var spec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"user_id": pagination.TypeUUID,
		"type":    pagination.TypeString,
		"channel": pagination.TypeString,
		"status":  pagination.TypeString,
	},
	map[string]pagination.ColumnType{
		"user_id": pagination.TypeUUID,
		"type":    pagination.TypeString,
		"channel": pagination.TypeString,
		"status":  pagination.TypeString,
	},
	"title", "body",
)

// Create adds a notification (queued as pending).
func (s *Service) Create(ctx context.Context, req CreateRequest, meta audit.Meta) (*View, error) {
	if req.UserID == uuid.Nil || strings.TrimSpace(req.Title) == "" {
		return nil, errors.New(400, "user_id and title are required")
	}
	row := models.Notification{
		UserID:  req.UserID,
		Title:   strings.TrimSpace(req.Title),
		Body:    req.Body,
		Type:    modkit.OrDefault(req.Type, "push"),
		Channel: modkit.OrDefault(req.Channel, "push"),
		Status:  models.NotificationPending,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	s.audit.Log(ctx, meta, "notification", row.ID.String(), "create", nil, &row)
	return toView(&row), nil
}

// MarkRead marks a notification read.
func (s *Service) MarkRead(ctx context.Context, req MarkReadRequest) (*View, error) {
	var row models.Notification
	if err := s.db.WithContext(ctx).First(&row, "id = ?", req.ID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&row).Update("read_at", &now).Error; err != nil {
		return nil, err
	}
	var after models.Notification
	s.db.WithContext(ctx).First(&after, "id = ?", row.ID)
	return toView(&after), nil
}

// List returns cursor-paged notifications with summary.
func (s *Service) List(ctx context.Context, req ListRequest) ([]View, *response.Pagination, *Summary, error) {
	items := []models.Notification{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.Notification{}), req.PageRequest, spec, &items)
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
	base := s.db.WithContext(ctx).Model(&models.Notification{})
	filtered, err := pagination.FilteredQuery(base, req, spec)
	if err != nil {
		return nil, err
	}
	var sums struct {
		Total   int64 `gorm:"column:total"`
		Pending int64 `gorm:"column:pending"`
		Sent    int64 `gorm:"column:sent"`
		Failed  int64 `gorm:"column:failed"`
		Unread  int64 `gorm:"column:unread"`
	}
	if err := filtered.Select(`
		COUNT(*) AS total,
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending,
		SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) AS sent,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed,
		SUM(CASE WHEN read_at IS NULL THEN 1 ELSE 0 END) AS unread`).Scan(&sums).Error; err != nil {
		return nil, err
	}
	return &Summary{Total: sums.Total, Pending: sums.Pending, Sent: sums.Sent, Failed: sums.Failed, Unread: sums.Unread}, nil
}

func toView(row *models.Notification) *View {
	return &View{
		ID:        row.ID.String(),
		UserID:    row.UserID.String(),
		Title:     row.Title,
		Body:      row.Body,
		Type:      row.Type,
		Channel:   row.Channel,
		Status:    row.Status,
		SentAt:    row.SentAt,
		ReadAt:    row.ReadAt,
		CreatedAt: row.CreatedAt,
	}
}
