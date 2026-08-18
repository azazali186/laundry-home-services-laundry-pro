package notification

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newHandlerEnv(t *testing.T) *Handler {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Notification{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewHandler(NewService(db, audit.NewWriter(db)))
}

func jsonBody(t *testing.T, v any) *ut.Body {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &ut.Body{Body: bytes.NewReader(b), Len: len(b)}
}

func jsonHeader() ut.Header { return ut.Header{Key: "Content-Type", Value: "application/json"} }

// createNotification seeds one and returns its id from the envelope.
func createNotification(t *testing.T, e *server.Hertz, user uuid.UUID) uuid.UUID {
	t.Helper()
	resp := ut.PerformRequest(e.Engine, http.MethodPost, "/notifications",
		jsonBody(t, CreateRequest{UserID: user, Title: "Order picked up"}), jsonHeader())
	if resp.Code != http.StatusCreated {
		t.Fatalf("create = %d, want 201 (body: %s)", resp.Code, resp.Body.String())
	}
	var env struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil || env.Data.ID == "" {
		t.Fatalf("create envelope missing id: %s", resp.Body.String())
	}
	id, err := uuid.Parse(env.Data.ID)
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}
	return id
}

func TestHandlerCreateNotification(t *testing.T) {
	h := newHandlerEnv(t)
	e := server.New(server.WithHostPorts("127.0.0.1:0"))
	e.POST("/notifications", h.Create)

	resp := ut.PerformRequest(e.Engine, http.MethodPost, "/notifications",
		jsonBody(t, CreateRequest{UserID: uuid.New(), Title: "Order picked up"}), jsonHeader())
	if resp.Code != http.StatusCreated {
		t.Fatalf("create = %d, want 201 (body: %s)", resp.Code, resp.Body.String())
	}
	if got := resp.Body.String(); !strings.Contains(got, "Order picked up") || !strings.Contains(got, `"code":0`) {
		t.Fatalf("create body missing data: %s", got)
	}

	// Validation: missing user id -> 400.
	resp = ut.PerformRequest(e.Engine, http.MethodPost, "/notifications",
		jsonBody(t, CreateRequest{Title: "No user"}), jsonHeader())
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("invalid create = %d, want 400", resp.Code)
	}
}

func TestHandlerMarkRead(t *testing.T) {
	h := newHandlerEnv(t)
	e := server.New(server.WithHostPorts("127.0.0.1:0"))
	e.POST("/notifications", h.Create)
	e.POST("/notifications/mark-read", h.MarkRead)

	user := uuid.New()
	id := createNotification(t, e, user)

	// Happy path: mark the created notification read.
	resp := ut.PerformRequest(e.Engine, http.MethodPost, "/notifications/mark-read",
		jsonBody(t, MarkReadRequest{ID: id}), jsonHeader())
	if resp.Code != http.StatusOK {
		t.Fatalf("mark-read = %d, want 200 (body: %s)", resp.Code, resp.Body.String())
	}
	if got := resp.Body.String(); !strings.Contains(got, `"read_at"`) {
		t.Fatalf("mark-read body missing read_at: %s", got)
	}

	// Unknown id -> 404.
	resp = ut.PerformRequest(e.Engine, http.MethodPost, "/notifications/mark-read",
		jsonBody(t, MarkReadRequest{ID: uuid.New()}), jsonHeader())
	if resp.Code != http.StatusNotFound {
		t.Fatalf("mark unknown = %d, want 404", resp.Code)
	}

	// Missing id -> 400.
	resp = ut.PerformRequest(e.Engine, http.MethodPost, "/notifications/mark-read",
		jsonBody(t, map[string]string{}), jsonHeader())
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("mark without id = %d, want 400", resp.Code)
	}
}

func TestHandlerListNotifications(t *testing.T) {
	h := newHandlerEnv(t)
	e := server.New(server.WithHostPorts("127.0.0.1:0"))
	e.POST("/notifications", h.Create)
	e.POST("/notifications/list", h.List)

	user := uuid.New()
	for i := 0; i < 3; i++ {
		createNotification(t, e, user)
	}

	resp := ut.PerformRequest(e.Engine, http.MethodPost, "/notifications/list",
		jsonBody(t, ListRequest{}), jsonHeader())
	if resp.Code != http.StatusOK {
		t.Fatalf("list = %d, want 200 (body: %s)", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"total":3`) || !strings.Contains(body, `"summary"`) {
		t.Fatalf("list body missing pagination/summary: %s", body)
	}
}
