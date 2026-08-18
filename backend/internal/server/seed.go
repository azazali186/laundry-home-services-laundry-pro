package server

import (
	"context"
	"errors"

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/crypto"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/password"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role seeds.
const (
	RoleSuperAdmin = "super_admin"
	RoleOpsManager = "ops_manager"
	RoleAgent      = "agent"
	RoleCustomer   = "customer"
)

// roleDefs are the bootstrap roles. Super admin receives a wildcard
// permission so every future route is automatically authorized.
var roleDefs = []struct {
	Name string
	Code string
	Desc string
}{
	{"Super Admin", RoleSuperAdmin, "Full system access"},
	{"Operations Manager", RoleOpsManager, "Manage orders, pickups, deliveries"},
	{"Delivery Agent", RoleAgent, "Handle pickups and deliveries"},
	{"Customer", RoleCustomer, "Self-service booking and tracking"},
}

// SeedBootstrap ensures system roles and the admin user exist (idempotent).
func SeedBootstrap(ctx context.Context, db *gorm.DB, cfg *config.Config, cipher *crypto.Cipher) error {
	superRole, err := ensureRole(ctx, db, roleDefs[0])
	if err != nil {
		return err
	}
	for _, def := range roleDefs[1:] {
		if _, err := ensureRole(ctx, db, def); err != nil {
			return err
		}
	}

	// Wildcard permission grants the admin access to every API route.
	wildcard := models.Permission{Name: "All Routes", Route: "*", Path: "*", Method: "*", Service: "api-gateway", Guard: "API"}
	if err := db.WithContext(ctx).Where("route = ?", "*").FirstOrCreate(&wildcard).Error; err != nil {
		return err
	}
	hasWildcard := db.WithContext(ctx).Model(&models.Role{Base: models.Base{ID: superRole.ID}}).
		Association("Permissions").Count()
	if hasWildcard == 0 {
		if err := db.WithContext(ctx).Model(&models.Role{Base: models.Base{ID: superRole.ID}}).
			Association("Permissions").Append(&wildcard); err != nil {
			return err
		}
	}

	return ensureAdmin(ctx, db, cfg, cipher, superRole.ID)
}

// ensureRole creates a role when missing.
func ensureRole(ctx context.Context, db *gorm.DB, def struct {
	Name string
	Code string
	Desc string
}) (*models.Role, error) {
	var role models.Role
	err := db.WithContext(ctx).Where("code = ?", def.Code).First(&role).Error
	if err == nil {
		return &role, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	role = models.Role{Name: def.Name, Code: def.Code, Description: def.Desc, IsSystem: true}
	if err := db.WithContext(ctx).Create(&role).Error; err != nil {
		return nil, err
	}
	logger.Info("role seeded: %s", def.Code)
	return &role, nil
}

// catalogSvcDefs are the bootstrap service types. Prices and units are the
// default catalog; admins manage them through the service-types module.
type catalogSvcDef struct {
	Code  string
	Name  string
	Unit  string
	Desc  string
	Icon  string
	Price float64
	Days  int
}

var catalogSvcDefs = []catalogSvcDef{
	{"WASH", "Wash & Fold", "kg", "Everyday loads washed, dried and neatly folded by weight.", "🧺", 99, 1},
	{"DRY", "Dry Cleaning", "per_item", "Silks, suits and delicates treated with solvent-safe care by certified finishers.", "👔", 149, 2},
	{"IRON", "Ironing & Pressing", "per_item", "Crisp, hotel-grade finishing on shirts, trousers and linens.", "☀️", 59, 1},
	{"LINEN", "Curtains & Home Linen", "per_item", "Heavy curtains, duvets and bedding cleaned at commercial scale, with fold-to-fit delivery.", "🏠", 199, 3},
	{"SHOES", "Shoe & Bag Care", "per_item", "Sneaker deep-cleaning and leather conditioning, returned boxed with fresh shoe trees.", "👟", 249, 3},
}

// catalogComboDefs are the bootstrap combos (priced bundles). Each entry maps
// a service-type code to a quantity.
type catalogComboDef struct {
	Code  string
	Name  string
	Unit  string
	Desc  string
	Price float64
	Items map[string]int
}

var catalogComboDefs = []catalogComboDef{
	{"WEEKLY", "Weekly Pass", "per_week", "Up to 4 loads (6 kg each) with free pickup, priority turnaround and photo confirmation on every bag.", 499, map[string]int{"WASH": 4}},
	{"FAMILY", "Family Plan", "per_month", "Up to 12 loads per month, bulk linen & curtain service and a dedicated driver for your area.", 1299, map[string]int{"WASH": 12, "LINEN": 1}},
}

// SeedCatalog ensures the default service types and combos exist (idempotent).
// Currency follows the shop settings row so the seeded catalog matches the
// configured business.
func SeedCatalog(ctx context.Context, db *gorm.DB) error {
	currency := "INR"
	var setting models.ShopSetting
	if err := db.WithContext(ctx).First(&setting).Error; err == nil && setting.Currency != "" {
		currency = setting.Currency
	}
	serviceIDs := map[string]uuid.UUID{}
	for _, d := range catalogSvcDefs {
		// Match by code OR name (both unique) so a pre-existing row created
		// under a different code never trips the name unique constraint.
		var row models.ServiceType
		err := db.WithContext(ctx).Where("code = ? OR lower(name) = lower(?)", d.Code, d.Name).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = models.ServiceType{Name: d.Name, Code: d.Code, Description: d.Desc, Unit: d.Unit, Price: d.Price, Currency: currency, EstimatedDays: d.Days, IsActive: true, Icon: d.Icon}
			if err := db.WithContext(ctx).Create(&row).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		serviceIDs[d.Code] = row.ID
	}
	for _, d := range catalogComboDefs {
		var row models.Combo
		err := db.WithContext(ctx).Where("code = ? OR lower(name) = lower(?)", d.Code, d.Name).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = models.Combo{Name: d.Name, Code: d.Code, Description: d.Desc, Unit: d.Unit, Price: d.Price, Currency: currency, IsActive: true}
			if err := db.WithContext(ctx).Create(&row).Error; err != nil {
				return err
			}
			for svcCode, qty := range d.Items {
				item := models.ComboItem{ComboID: row.ID, ServiceTypeID: serviceIDs[svcCode], Quantity: qty}
				if err := db.WithContext(ctx).Create(&item).Error; err != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		}
	}
	logger.Info("catalog seeded: %d service types, %d combos", len(catalogSvcDefs), len(catalogComboDefs))
	return nil
}

// ensureAdmin creates the configured admin user with the super admin role.
func ensureAdmin(ctx context.Context, db *gorm.DB, cfg *config.Config, cipher *crypto.Cipher, superRoleID uuid.UUID) error {
	var user models.User
	err := db.WithContext(ctx).Where("email = ?", cfg.AdminEmail).First(&user).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	hashed, err := password.Hash(cfg.AdminPassword, cfg.PasswordCost)
	if err != nil {
		return err
	}
	user = models.User{
		Name:     cfg.AdminName,
		Email:    cfg.AdminEmail,
		Password: hashed,
		IsActive: true,
	}
	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Model(&user).Association("Roles").Append(&models.Role{Base: models.Base{ID: superRoleID}}); err != nil {
		return err
	}
	logger.Info("admin user seeded: %s", cfg.AdminEmail)
	return nil
}
