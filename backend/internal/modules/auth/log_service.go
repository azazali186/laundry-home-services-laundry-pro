package auth

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"gorm.io/gorm"
)

// LogService serves login and audit log listings.
type LogService struct {
	db *gorm.DB
}

// NewLogService builds the log service.
func NewLogService(db *gorm.DB) *LogService { return &LogService{db: db} }

var loginLogSpec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"email":  pagination.TypeString,
		"status": pagination.TypeString,
		"ip":     pagination.TypeString,
	},
	map[string]pagination.ColumnType{
		"email":  pagination.TypeString,
		"status": pagination.TypeString,
		"ip":     pagination.TypeString,
	},
	"email",
)

// ListLoginLogs returns cursor-paged login attempts.
func (s *LogService) ListLoginLogs(ctx context.Context, req ListLoginLogsRequest) ([]LoginLogView, *response.Pagination, error) {
	items := []models.LoginLog{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.LoginLog{}), req.PageRequest, loginLogSpec, &items)
	if err != nil {
		return nil, nil, err
	}
	views := make([]LoginLogView, 0, len(items))
	for _, row := range items {
		views = append(views, LoginLogView{
			ID:        row.ID.String(),
			Email:     row.Email,
			IP:        row.IP,
			UserAgent: row.UserAgent,
			Status:    row.Status,
			Reason:    row.Reason,
			CreatedAt: row.CreatedAt,
		})
	}
	return views, pageResponse(res), nil
}

var auditLogSpec = pagination.NewSpec(
	map[string]pagination.ColumnType{
		"entity":    pagination.TypeString,
		"action":    pagination.TypeString,
		"entity_id": pagination.TypeString,
		"user_id":   pagination.TypeUUID,
	},
	map[string]pagination.ColumnType{
		"entity":    pagination.TypeString,
		"action":    pagination.TypeString,
		"entity_id": pagination.TypeString,
		"user_id":   pagination.TypeUUID,
	},
	"entity", "entity_id",
)

// ListAuditLogs returns cursor-paged audit entries.
func (s *LogService) ListAuditLogs(ctx context.Context, req ListAuditLogsRequest) ([]AuditLogView, *response.Pagination, error) {
	items := []models.AuditLog{}
	res, err := pagination.Page(s.db.WithContext(ctx).Model(&models.AuditLog{}), req.PageRequest, auditLogSpec, &items)
	if err != nil {
		return nil, nil, err
	}
	views := make([]AuditLogView, 0, len(items))
	for _, row := range items {
		views = append(views, AuditLogView{
			ID:        row.ID.String(),
			UserID:    row.UserID,
			Entity:    row.Entity,
			EntityID:  row.EntityID,
			Action:    row.Action,
			Method:    row.Method,
			Path:      row.Path,
			IP:        row.IP,
			Before:    row.Before,
			After:     row.After,
			CreatedAt: row.CreatedAt,
		})
	}
	return views, pageResponse(res), nil
}
