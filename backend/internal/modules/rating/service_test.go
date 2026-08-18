package rating

import (
	"context"
	"math"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
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
	if err := db.AutoMigrate(&models.Rating{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, NewService(db, audit.NewWriter(db))
}

var testMeta = audit.Meta{UserID: uuid.New(), Method: "POST", Path: "/api/v1/ratings"}

func TestRatingCreateAndValidation(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Score: 5, Comment: "Great"}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Score != 5 {
		t.Fatalf("score = %d, want 5", view.Score)
	}
	if _, err := svc.Create(ctx, CreateRequest{Score: 4}, testMeta); err == nil {
		t.Fatal("missing order_id must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Score: 0}, testMeta); err == nil {
		t.Fatal("score below 1 must fail")
	}
	if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Score: 6}, testMeta); err == nil {
		t.Fatal("score above 5 must fail")
	}
}

func TestRatingUpdateValidatesScore(t *testing.T) {
	db, svc := newEnv(t)
	ctx := context.Background()
	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Score: 3}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	bad := 9
	if _, err := svc.Update(ctx, UpdateRequest{ID: id, Score: &bad}, testMeta); err == nil {
		t.Fatal("out-of-range score must fail")
	}
	good := 4
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, Score: &good}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Score != 4 {
		t.Fatalf("score = %d, want 4", updated.Score)
	}

	if err := svc.Delete(ctx, IDRequest{ID: id}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Rating{}).Count(&count)
	if count != 0 {
		t.Fatalf("rows after delete = %d, want 0", count)
	}
}

func TestRatingListAndSummary(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()

	// One rating per score bucket: 5,5,4,2,1 -> avg 3.4.
	for _, sc := range []int{5, 5, 4, 2, 1} {
		if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Score: sc}, testMeta); err != nil {
			t.Fatalf("Create(%d): %v", sc, err)
		}
	}

	items, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.Total != 5 {
		t.Fatalf("pagination total = %d, want 5", pg.Total)
	}
	if len(items) != 2 {
		t.Fatalf("page size = %d, want 2", len(items))
	}
	if summary.Total != 5 {
		t.Fatalf("summary total = %d, want 5", summary.Total)
	}
	if math.Abs(summary.AvgScore-3.4) > 0.001 {
		t.Fatalf("avg score = %v, want 3.4", summary.AvgScore)
	}
	if summary.FiveStar != 2 || summary.FourStar != 1 || summary.ThreeStar != 0 || summary.TwoStar != 1 || summary.OneStar != 1 {
		t.Fatalf("star breakdown wrong: %+v", summary)
	}
}

func TestRatingListFilterAndSearch(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()

	for _, sc := range []int{5, 5, 4, 3} {
		if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Score: sc}, testMeta); err != nil {
			t.Fatalf("Create(%d): %v", sc, err)
		}
	}

	// Filter narrows both the page and the summary.
	items, pg, summary, err := svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{
		Filters: map[string]string{"score": "5"},
	}})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if pg.Total != 2 || len(items) != 2 {
		t.Fatalf("filtered total/len = %d/%d, want 2/2", pg.Total, len(items))
	}
	if summary.Total != 2 || summary.FiveStar != 2 {
		t.Fatalf("filtered summary wrong: %+v", summary)
	}

	// Search across the searchable column (comment).
	if _, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Score: 4, Comment: "exceptional service"}, testMeta); err != nil {
		t.Fatalf("Create searchable: %v", err)
	}
	_, pg, _, err = svc.List(ctx, ListRequest{PageRequest: pagination.PageRequest{Search: "exceptional"}})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if pg.Total != 1 {
		t.Fatalf("search total = %d, want 1", pg.Total)
	}
}

func TestRatingUpdateCommentAndPublic(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()

	view, err := svc.Create(ctx, CreateRequest{OrderID: uuid.New(), Score: 3, Comment: "old", IsPublic: false}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	id, _ := uuid.Parse(view.ID)

	comment := "updated"
	pub := true
	updated, err := svc.Update(ctx, UpdateRequest{ID: id, Comment: &comment, IsPublic: &pub}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Comment != "updated" || !updated.IsPublic {
		t.Fatalf("comment/public not updated: %+v", updated)
	}
	if updated.Score != 3 {
		t.Fatalf("score changed unexpectedly: %+v", updated)
	}
}

func TestRatingNotFound(t *testing.T) {
	_, svc := newEnv(t)
	ctx := context.Background()
	missing := uuid.New()

	if _, err := svc.Update(ctx, UpdateRequest{ID: missing}, testMeta); err == nil {
		t.Fatal("update of missing rating must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("update missing: %v", err)
	}
	if err := svc.Delete(ctx, IDRequest{ID: missing}, testMeta); err == nil {
		t.Fatal("delete of missing rating must fail")
	} else if e := errors.As(err); e == nil || e.Code != 404 {
		t.Fatalf("delete missing: %v", err)
	}
}
