package audit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAuditDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestRedactor(t *testing.T) {
	in := []byte(`{"name":"Ada","password":"hunter2","token":"jwt-abc","card_last4":"4242","cvv":"123","keep":"ok"}`)
	out := Redactor(in)

	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal redacted: %v", err)
	}
	for _, key := range []string{"password", "token", "card_last4", "cvv"} {
		if m[key] != "[redacted]" {
			t.Errorf("%s not redacted: %v", key, m[key])
		}
	}
	if m["name"] != "Ada" || m["keep"] != "ok" {
		t.Fatalf("non-sensitive data was altered: %v", m)
	}
	if strings.Contains(string(out), "hunter2") || strings.Contains(string(out), "jwt-abc") {
		t.Fatal("sensitive value leaked into redacted output")
	}
}

func TestRedactorEdgeCases(t *testing.T) {
	if Redactor(nil) != nil {
		t.Fatal("Redactor(nil) must return nil")
	}
	// Non-JSON input passes through untouched.
	raw := []byte("not json")
	if got := Redactor(raw); string(got) != "not json" {
		t.Fatalf("Redactor(non-json) = %s", got)
	}
}

func TestRedactorCaseInsensitive(t *testing.T) {
	out := Redactor([]byte(`{"Password":"x","PASSWORD":"y"}`))
	if !strings.Contains(string(out), "[redacted]") {
		t.Fatal("sensitive keys must be matched case-insensitively")
	}
}

func TestChangedFields(t *testing.T) {
	before := map[string]any{"name": "Ada", "price": 100, "password": "old"}
	after := map[string]any{"name": "Grace", "price": 100, "password": "new"}
	changed := ChangedFields(before, after)

	if len(changed) != 1 || changed[0] != "name" {
		t.Fatalf("changed = %v, want [name] only (password must be skipped)", changed)
	}
}

func TestLogPersistsRedactedSnapshots(t *testing.T) {
	db := newAuditDB(t)
	w := NewWriter(db)

	type user struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	w.Log(context.Background(),
		Meta{UserID: uuid.New(), IP: "10.0.0.1", Method: "PATCH", Path: "/api/v1/users"},
		"user", "u-1", "update",
		user{Name: "Old", Password: "hunter2"},
		user{Name: "New", Password: "hunter3"},
	)

	var row models.AuditLog
	if err := db.First(&row).Error; err != nil {
		t.Fatalf("audit row missing: %v", err)
	}
	if row.Entity != "user" || row.Action != "update" || row.IP != "10.0.0.1" {
		t.Fatalf("unexpected row: %+v", row)
	}
	if strings.Contains(string(row.Before), "hunter2") || strings.Contains(string(row.After), "hunter3") {
		t.Fatal("sensitive values leaked into the audit snapshots")
	}
	if !strings.Contains(string(row.After), "[redacted]") {
		t.Fatal("after snapshot should contain the redaction marker")
	}
}

func TestLogNeverBreaksOnDBFailure(t *testing.T) {
	db := newAuditDB(t)
	// Close the DB so the Create fails — Log must swallow the error.
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	w := NewWriter(db)
	w.Log(context.Background(), Meta{}, "user", "u-1", "create", nil, map[string]any{"name": "x"})
}
