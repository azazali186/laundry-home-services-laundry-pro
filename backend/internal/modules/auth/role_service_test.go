package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newRoleEnv(t *testing.T) (*gorm.DB, *RoleService) {
	t.Helper()
	dsn := "file:" + uuid.New().String() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Role{}, &models.Permission{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewFromClient(client, "test")
	svc := NewRoleService(db, audit.NewWriter(db), NewRbacService(db, c))
	return db, svc
}

func seedPermission(t *testing.T, db *gorm.DB, name, route string) models.Permission {
	t.Helper()
	p := models.Permission{Name: name, Route: route, Path: "/api/v1/x", Method: "POST", Service: "api-gateway", Guard: "API"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}
	return p
}

func TestRoleCreateValidatesAndAssignsPermissions(t *testing.T) {
	db, svc := newRoleEnv(t)
	ctx := context.Background()
	perm := seedPermission(t, db, "Create Booking", "POST /api/v1/bookings")

	if _, err := svc.Create(ctx, CreateRoleRequest{Name: "", Code: "x"}, testMeta); errors.As(err) == nil || errors.As(err).Code != 400 {
		t.Fatalf("missing name: want 400, got %v", err)
	}

	view, err := svc.Create(ctx, CreateRoleRequest{
		Name: "Booking Manager", Code: "  Booking_Mgr ",
		PermissionIDs: []uuid.UUID{perm.ID},
	}, testMeta)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if view.Code != "booking_mgr" {
		t.Fatalf("code not normalized: %q", view.Code)
	}
	if len(view.Permissions) != 1 || view.Permissions[0].ID != perm.ID.String() {
		t.Fatalf("permissions not assigned: %+v", view.Permissions)
	}

	// Duplicate code -> 409.
	if _, err := svc.Create(ctx, CreateRoleRequest{Name: "Dup", Code: "booking_mgr"}, testMeta); errors.As(err) == nil || errors.As(err).Code != 409 {
		t.Fatalf("duplicate code: want 409, got %v", err)
	}
}

func TestRoleUpdateBlocksSystemRenameAndReplacesPermissions(t *testing.T) {
	db, svc := newRoleEnv(t)
	ctx := context.Background()
	permA := seedPermission(t, db, "A", "POST /api/v1/a")
	permB := seedPermission(t, db, "B", "POST /api/v1/b")

	systemRole := models.Role{Name: "Super Admin", Code: "super_admin", IsSystem: true}
	if err := db.Create(&systemRole).Error; err != nil {
		t.Fatalf("create system role: %v", err)
	}
	rename := "Hacked"
	if _, err := svc.Update(ctx, UpdateRoleRequest{ID: systemRole.ID, Name: &rename}, testMeta); errors.As(err) == nil || errors.As(err).Code != 400 {
		t.Fatalf("renaming system role: want 400, got %v", err)
	}

	role, err := svc.Create(ctx, CreateRoleRequest{Name: "Ops", Code: "ops", PermissionIDs: []uuid.UUID{permA.ID}}, testMeta)
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	newName := "Ops Manager"
	roleID, err := uuid.Parse(role.ID)
	if err != nil {
		t.Fatalf("parse role id: %v", err)
	}
	view, err := svc.Update(ctx, UpdateRoleRequest{ID: roleID, Name: &newName, PermissionIDs: []uuid.UUID{permB.ID}}, testMeta)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if view.Name != "Ops Manager" {
		t.Fatalf("name not updated: %q", view.Name)
	}
	if len(view.Permissions) != 1 || view.Permissions[0].ID != permB.ID.String() {
		t.Fatalf("permissions not replaced: %+v", view.Permissions)
	}
}

func TestRoleDeleteAndAssignPermissions(t *testing.T) {
	db, svc := newRoleEnv(t)
	ctx := context.Background()
	perm := seedPermission(t, db, "P", "POST /api/v1/p")

	systemRole := models.Role{Name: "Super Admin", Code: "super_admin", IsSystem: true}
	if err := db.Create(&systemRole).Error; err != nil {
		t.Fatalf("create system role: %v", err)
	}
	if err := svc.Delete(ctx, DeleteRoleRequest{ID: systemRole.ID}, testMeta); errors.As(err) == nil || errors.As(err).Code != 400 {
		t.Fatalf("deleting system role: want 400, got %v", err)
	}

	role, err := svc.Create(ctx, CreateRoleRequest{Name: "Temp", Code: "temp"}, testMeta)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	roleID, err := uuid.Parse(role.ID)
	if err != nil {
		t.Fatalf("parse role id: %v", err)
	}
	view, err := svc.AssignPermissions(ctx, AssignPermissionsRequest{RoleID: roleID, PermissionIDs: []uuid.UUID{perm.ID}}, testMeta)
	if err != nil {
		t.Fatalf("AssignPermissions: %v", err)
	}
	if len(view.Permissions) != 1 {
		t.Fatalf("assign failed: %+v", view.Permissions)
	}
	if err := svc.Delete(ctx, DeleteRoleRequest{ID: roleID}, testMeta); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	db.Model(&models.Role{}).Where("id = ?", roleID).Count(&count)
	if count != 0 {
		t.Fatal("role not deleted")
	}
}

func TestRoleListAndPermissionList(t *testing.T) {
	db, svc := newRoleEnv(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		code := fmt.Sprintf("role_%d", i)
		if _, err := svc.Create(ctx, CreateRoleRequest{Name: "Role " + code, Code: code}, testMeta); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	seedPermission(t, db, "P1", "POST /api/v1/1")
	seedPermission(t, db, "P2", "POST /api/v1/2")

	roles, pg, err := svc.List(ctx, ListRolesRequest{PageRequest: pagination.PageRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(roles) != 2 || pg.Total != 3 {
		t.Fatalf("roles page = %d total %d, want 2/3", len(roles), pg.Total)
	}
	perms, ppg, err := svc.ListPermissions(ctx, ListPermissionsRequest{PageRequest: pagination.PageRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("ListPermissions: %v", err)
	}
	if len(perms) != 2 || ppg.Total != 2 {
		t.Fatalf("perms page = %d total %d, want 2/2", len(perms), ppg.Total)
	}
}
