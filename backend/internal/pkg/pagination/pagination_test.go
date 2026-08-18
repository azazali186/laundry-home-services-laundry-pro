package pagination

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testRow is a minimal table for exercising the pager.
type testRow struct {
	ID        uuid.UUID `gorm:"type:text;primaryKey" json:"id"`
	Name      string    `gorm:"size:80" json:"name"`
	Score     int       `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}

func (testRow) TableName() string { return "test_rows" }

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&testRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedRows(t *testing.T, db *gorm.DB, n int) {
	t.Helper()
	base := time.Now().Add(-time.Duration(n) * time.Minute)
	for i := 0; i < n; i++ {
		row := testRow{
			ID:        uuid.New(),
			Name:      fmt.Sprintf("item-%02d", i),
			Score:     i * 10,
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
}

func rowSpec() Spec {
	return NewSpec(
		map[string]ColumnType{"score": TypeNumber, "name": TypeString},
		map[string]ColumnType{"score": TypeNumber, "name": TypeString},
		"name",
	)
}

func TestPageBasics(t *testing.T) {
	db := newTestDB(t)
	seedRows(t, db, 25)

	ctx := context.Background()
	base := db.WithContext(ctx).Model(&testRow{})

	// Page 1
	var page1 []testRow
	res1, err := Page(base, PageRequest{Limit: 10, Offset: 0}, rowSpec(), &page1)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 10 {
		t.Fatalf("page1 len = %d, want 10", len(page1))
	}
	if res1.Total != 25 {
		t.Fatalf("total = %d, want 25", res1.Total)
	}
	if !res1.HasMore {
		t.Fatal("page1 should have more")
	}
	if res1.Offset != 0 {
		t.Fatalf("page1 offset = %d, want 0", res1.Offset)
	}

	// Page 2
	var page2 []testRow
	res2, err := Page(base, PageRequest{Limit: 10, Offset: 10}, rowSpec(), &page2)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 10 {
		t.Fatalf("page2 len = %d, want 10", len(page2))
	}
	if res2.Offset != 10 {
		t.Fatalf("page2 offset = %d, want 10", res2.Offset)
	}

	// Page 3 (partial)
	var page3 []testRow
	res3, err := Page(base, PageRequest{Limit: 10, Offset: 20}, rowSpec(), &page3)
	if err != nil {
		t.Fatalf("page3: %v", err)
	}
	if len(page3) != 5 {
		t.Fatalf("page3 len = %d, want 5", len(page3))
	}
	if res3.HasMore {
		t.Fatal("page3 should be the final page")
	}
	if res3.Offset != 20 {
		t.Fatalf("page3 offset = %d, want 20", res3.Offset)
	}

	// No overlaps across pages.
	seen := map[string]bool{}
	for _, rows := range [][]testRow{page1, page2, page3} {
		for _, r := range rows {
			if seen[r.ID.String()] {
				t.Fatalf("duplicate row %s across pages", r.ID)
			}
			seen[r.ID.String()] = true
		}
	}
	if len(seen) != 25 {
		t.Fatalf("saw %d unique rows, want 25", len(seen))
	}
}

func TestPageSortAsc(t *testing.T) {
	db := newTestDB(t)
	seedRows(t, db, 25)

	ctx := context.Background()
	base := db.WithContext(ctx).Model(&testRow{})

	var rows []testRow
	_, err := Page(base, PageRequest{Limit: 5, Offset: 0, SortBy: "score", SortOrder: "asc"}, rowSpec(), &rows)
	if err != nil {
		t.Fatalf("page: %v", err)
	}
	if rows[0].Score != 0 || rows[4].Score != 40 {
		t.Fatalf("asc scores wrong: %+v", rows)
	}
}

func TestPageFiltersAndSearch(t *testing.T) {
	db := newTestDB(t)
	seedRows(t, db, 25)

	ctx := context.Background()
	base := db.WithContext(ctx).Model(&testRow{})

	// Numeric range filter.
	var rows []testRow
	res, err := Page(base, PageRequest{Limit: 100, Filters: map[string]string{"score": "gte:100"}}, rowSpec(), &rows)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if res.Total != 15 {
		t.Fatalf("filter total = %d, want 15", res.Total)
	}
	if len(rows) != 15 {
		t.Fatalf("filter rows = %d, want 15", len(rows))
	}

	// Search across the name column.
	var found []testRow
	res, err = Page(base, PageRequest{Limit: 100, Search: "item-1"}, rowSpec(), &found)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if res.Total == 0 || int(res.Total) != len(found) {
		t.Fatalf("search total = %d rows = %d", res.Total, len(found))
	}
	for _, r := range found {
		if r.Name != "item-1" && r.Name != "item-10" && r.Name != "item-11" && r.Name != "item-12" &&
			r.Name != "item-13" && r.Name != "item-14" && r.Name != "item-15" && r.Name != "item-16" &&
			r.Name != "item-17" && r.Name != "item-18" && r.Name != "item-19" {
			t.Fatalf("unexpected search hit: %s", r.Name)
		}
	}
}

func TestPageRejectsBadInput(t *testing.T) {
	db := newTestDB(t)
	seedRows(t, db, 5)
	ctx := context.Background()
	base := db.WithContext(ctx).Model(&testRow{})

	var rows []testRow

	// Unknown sort column must be rejected (injection guard).
	if _, err := Page(base, PageRequest{SortBy: "id; DROP TABLE test_rows"}, rowSpec(), &rows); err == nil {
		t.Fatal("malicious sort column was accepted")
	}
	// Unknown filter column must be rejected.
	if _, err := Page(base, PageRequest{Filters: map[string]string{"password": "x"}}, rowSpec(), &rows); err == nil {
		t.Fatal("unknown filter column was accepted")
	}
	// Invalid sort order must be rejected.
	if _, err := Page(base, PageRequest{SortOrder: "sideways"}, rowSpec(), &rows); err == nil {
		t.Fatal("invalid sort order was accepted")
	}
}

func TestNegativeOffsetNormalized(t *testing.T) {
	db := newTestDB(t)
	seedRows(t, db, 5)
	ctx := context.Background()
	base := db.WithContext(ctx).Model(&testRow{})

	var rows []testRow
	res, err := Page(base, PageRequest{Offset: -10, Limit: 5}, rowSpec(), &rows)
	if err != nil {
		t.Fatalf("negative offset: %v", err)
	}
	// Negative offset is normalized to 0.
	if res.Offset != 0 {
		t.Fatalf("offset = %d, want 0", res.Offset)
	}
	if len(rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(rows))
	}
}

func asAppError(err error, out **errors.AppError) bool {
	o, ok := err.(*errors.AppError)
	if ok {
		*out = o
	}
	return ok
}
