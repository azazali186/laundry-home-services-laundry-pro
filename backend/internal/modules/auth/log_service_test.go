package auth

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newLogEnv(t *testing.T) (*gorm.DB, *LogService) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.LoginLog{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, NewLogService(db)
}

func TestListLoginLogsPaginates(t *testing.T) {
	db, svc := newLogEnv(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		row := models.LoginLog{
			UserID: uuid.New(), Email: "u@b.com", IP: "10.0.0.1",
			UserAgent: "curl", Status: "failed", Reason: "invalid password",
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed login log: %v", err)
		}
	}
	rows, pg, err := svc.ListLoginLogs(ctx, ListLoginLogsRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("ListLoginLogs: %v", err)
	}
	if len(rows) != 2 || pg.Total != 3 {
		t.Fatalf("login log page = %d total %d, want 2/3", len(rows), pg.Total)
	}
	if rows[0].Email != "u@b.com" || rows[0].Status != "failed" {
		t.Fatalf("unexpected login log view: %+v", rows[0])
	}
}

func TestListAuditLogsPaginates(t *testing.T) {
	db, svc := newLogEnv(t)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		row := models.AuditLog{
			Base:   models.Base{CreatedAt: time.Now()},
			UserID: uuid.New(), Entity: "user", EntityID: uuid.New().String(),
			Action: "update", Method: "PATCH", Path: "/api/v1/users",
			IP: "10.0.0.1", Before: []byte(`{"a":1}`), After: []byte(`{"a":2}`),
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed audit log: %v", err)
		}
	}
	rows, pg, err := svc.ListAuditLogs(ctx, ListAuditLogsRequest{PageRequest: pagination.PageRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	if len(rows) != 2 || pg.Total != 2 {
		t.Fatalf("audit log page = %d total %d, want 2/2", len(rows), pg.Total)
	}
	if rows[0].Action != "update" || rows[0].Method != "PATCH" {
		t.Fatalf("unexpected audit view: %+v", rows[0])
	}
}
