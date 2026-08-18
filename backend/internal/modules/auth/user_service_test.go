package auth

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/crypto"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/password"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newUserEnv(t *testing.T) (*gorm.DB, *UserService, *cache.Cache) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Role{}, &models.Permission{},
		&models.LoginLog{}, &models.AuditLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewFromClient(client, "test")
	cipher, err := crypto.NewCipher("test-encryption-key-0123456789")
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	cfg := &config.Config{PasswordCost: 4}
	auditWriter := audit.NewWriter(db)
	svc := NewUserService(db, c, cipher, cfg, auditWriter, NewRbacService(db, c))
	return db, svc, c
}

func seedRole(t *testing.T, db *gorm.DB, code string) models.Role {
	t.Helper()
	role := models.Role{Name: "Role " + code, Code: code, IsSystem: false}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	return role
}

func TestUserCreateValidatesAndAssignsRoles(t *testing.T) {
	db, svc, _ := newUserEnv(t)
	ctx := context.Background()
	role := seedRole(t, db, "agent")

	// Validation: missing name.
	if _, err := svc.Create(ctx, CreateUserRequest{Name: "", Email: "a@b.com", Password: "Secret123"}, testMeta); err == nil {
		t.Fatal("missing name must fail")
	}
	// Validation: short password.
	if _, err := svc.Create(ctx, CreateUserRequest{Name: "A", Email: "a@b.com", Password: "short"}, testMeta); err == nil {
		t.Fatal("short password must fail")
	}

	view, err := svc.Create(ctx, CreateUserRequest{
		Name: "Alice", Email: "  ALICE@b.com ", Phone: "+911234567890",
		Password: "Secret123", RoleIDs: []uuid.UUID{role.ID},
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Email != "alice@b.com" {
		t.Fatalf("email not normalized: %q", view.Email)
	}
	if view.Phone != "+911234567890" {
		t.Fatalf("phone not round-tripped through encryption: %q", view.Phone)
	}
	if len(view.Roles) != 1 || view.Roles[0].Code != "agent" {
		t.Fatalf("roles not assigned: %+v", view.Roles)
	}
	// The stored phone must be ciphertext (no sensitive data at rest).
	var row models.User
	if err := db.First(&row, "id = ?", view.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Phone == "+911234567890" {
		t.Fatal("phone stored in plaintext — sensitive data must be encrypted")
	}

	// Duplicate email.
	if _, err := svc.Create(ctx, CreateUserRequest{Name: "A2", Email: "alice@b.com", Password: "Secret123"}, testMeta); codeOf(err) != 409 {
		t.Fatalf("duplicate email: want 409, got %v", err)
	}
}

// codeOf extracts the app error code (0 when err is not an app error).
func codeOf(err error) int {
	if e := errors.As(err); e != nil {
		return e.Code
	}
	return 0
}

func TestUserUpdateFieldsRolesAndPassword(t *testing.T) {
	db, svc, _ := newUserEnv(t)
	ctx := context.Background()
	_ = seedRole(t, db, "agent")
	managerRole := seedRole(t, db, "ops_manager")

	u := seedUser(t, db, "u@b.com", "Secret123", true)
	if _, err := svc.Update(ctx, UpdateUserRequest{ID: uuid.New()}, testMeta); codeOf(err) != 404 {
		t.Fatalf("unknown user: want 404, got %v", err)
	}

	name := "Renamed"
	active := false
	view, err := svc.Update(ctx, UpdateUserRequest{
		ID: u.ID, Name: &name, IsActive: &active,
		RoleIDs: []uuid.UUID{managerRole.ID},
	}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if view.Name != "Renamed" || view.IsActive {
		t.Fatalf("fields not applied: %+v", view)
	}
	if len(view.Roles) != 1 || view.Roles[0].Code != "ops_manager" {
		t.Fatalf("roles not replaced: %+v", view.Roles)
	}

	// Password rotation.
	newPw := "Rotated123"
	if _, err := svc.Update(ctx, UpdateUserRequest{ID: u.ID, Password: &newPw}, testMeta); err != nil {
		t.Fatalf("Update password: %v", err)
	}
	var row models.User
	if err := db.First(&row, "id = ?", u.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !password.Verify(row.Password, "Rotated123") {
		t.Fatal("password not rotated")
	}

	// Roles cleared via an empty set replacement.
	if _, err := svc.Update(ctx, UpdateUserRequest{ID: u.ID, RoleIDs: []uuid.UUID{}}, testMeta); err != nil {
		t.Fatalf("Update clear roles: %v", err)
	}
	var after models.User
	if err := db.Preload("Roles").First(&after, "id = ?", u.ID).Error; err != nil {
		t.Fatalf("reload roles: %v", err)
	}
	if len(after.Roles) != 0 {
		t.Fatalf("roles not cleared: %+v", after.Roles)
	}
}

func TestUserDeleteSoftDeletesAndClearsSession(t *testing.T) {
	db, svc, c := newUserEnv(t)
	ctx := context.Background()
	u := seedUser(t, db, "del@b.com", "Secret123", true)
	if err := c.Set(ctx, "session:"+u.ID.String(), "fp", time.Hour); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	if err := svc.Delete(ctx, DeleteUserRequest{ID: uuid.New()}, testMeta); codeOf(err) != 404 {
		t.Fatalf("unknown user: want 404, got %v", err)
	}
	if err := svc.Delete(ctx, DeleteUserRequest{ID: u.ID}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var row models.User
	if err := db.Unscoped().First(&row, "id = ?", u.ID).Error; err != nil {
		t.Fatalf("row gone entirely: %v", err)
	}
	if !row.DeletedAt.Valid {
		t.Fatal("expected a soft delete (deleted_at set)")
	}
	if _, err := c.Get(ctx, "session:"+u.ID.String()); err == nil {
		t.Fatal("session not cleared after delete")
	}
	var auditRow models.AuditLog
	if err := db.First(&auditRow, "action = ?", "delete").Error; err != nil {
		t.Fatalf("audit delete entry missing: %v", err)
	}
}

func TestUserListPaginationAndSummary(t *testing.T) {
	db, svc, _ := newUserEnv(t)
	ctx := context.Background()
	superRole := models.Role{Name: "Super Admin", Code: "super_admin", IsSystem: true}
	if err := db.Create(&superRole).Error; err != nil {
		t.Fatalf("create super role: %v", err)
	}
	admin := seedUser(t, db, "admin@b.com", "Secret123", true)
	if err := db.Model(&admin).Association("Roles").Append(&superRole); err != nil {
		t.Fatalf("assign super role: %v", err)
	}
	seedUser(t, db, "b@b.com", "Secret123", true)
	seedUser(t, db, "off@b.com", "Secret123", false)

	views, pg, summary, err := svc.List(ctx, ListUsersRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("page size = %d, want 2", len(views))
	}
	if pg == nil || pg.Total != 3 {
		t.Fatalf("pagination total = %+v, want 3", pg)
	}
	if summary == nil {
		t.Fatal("summary missing")
	}
	if summary.TotalActive != 2 || summary.TotalInactive != 1 {
		t.Fatalf("summary active/inactive = %d/%d, want 2/1", summary.TotalActive, summary.TotalInactive)
	}
	if summary.TotalAdmins != 1 {
		t.Fatalf("summary admins = %d, want 1", summary.TotalAdmins)
	}

	// Search narrows results.
	_, pg, _, err = svc.List(ctx, ListUsersRequest{PageRequest: pagination.PageRequest{Search: "off@"}})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if pg.Total != 1 {
		t.Fatalf("search total = %d, want 1", pg.Total)
	}
}
