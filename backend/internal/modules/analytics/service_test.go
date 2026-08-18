package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newEnv(t *testing.T) (*gorm.DB, *Service) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Order{}, &models.Booking{}, &models.Pickup{},
		&models.Delivery{}, &models.Garment{}, &models.Invoice{}, &models.InventoryItem{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, NewService(db)
}

func TestDashboardAggregatesOrders(t *testing.T) {
	db, svc := newEnv(t)
	ctx := context.Background()
	now := time.Now()

	orders := []models.Order{
		{Base: models.Base{CreatedAt: now}, OrderNo: "O1", CustomerID: uuid.New(), ServiceTypeID: uuid.New(), Status: models.OrderDelivered, TotalAmount: 100, GrandTotal: 112, Currency: "INR"},
		{Base: models.Base{CreatedAt: now}, OrderNo: "O2", CustomerID: uuid.New(), ServiceTypeID: uuid.New(), Status: models.OrderProcessing, TotalAmount: 50, GrandTotal: 56, Currency: "INR"},
	}
	for i := range orders {
		if err := db.Create(&orders[i]).Error; err != nil {
			t.Fatalf("seed order: %v", err)
		}
	}

	dash, err := svc.Dashboard(ctx, DashboardRequest{
		From: now.AddDate(0, 0, -1).Format("2006-01-02"),
		To:   now.AddDate(0, 0, 1).Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}
	if dash.Period.Orders != 2 {
		t.Fatalf("orders = %d, want 2", dash.Period.Orders)
	}
	if dash.Period.Revenue != 168 {
		t.Fatalf("revenue = %v, want 168", dash.Period.Revenue)
	}
	if dash.Period.AvgOrder != 84 {
		t.Fatalf("avg order = %v, want 84", dash.Period.AvgOrder)
	}
	if len(dash.OrdersByStatus) != 2 {
		t.Fatalf("orders_by_status = %v, want 2 buckets", dash.OrdersByStatus)
	}
	if len(dash.RevenueTrend) == 0 {
		t.Fatal("revenue trend empty")
	}
}

func TestDashboardDefaultsToLastMonth(t *testing.T) {
	_, svc := newEnv(t)
	// Invalid range strings fall back to the last-month window instead of 500.
	dash, err := svc.Dashboard(context.Background(), DashboardRequest{From: "garbage", To: "nope"})
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}
	if dash.Period.From == "" || dash.Period.To == "" {
		t.Fatal("period must be populated")
	}
}
