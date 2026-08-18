// Package container wires all dependencies using explicit construction
// (no reflection): config -> infra -> services -> handlers.
package container

import (
	"context"
	"fmt"

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/flows"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/analytics"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/auth"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/booking"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/combo"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/delivery"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/garment"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/inventory"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/invoice"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/notification"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/order"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/payment"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/pickup"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/pricing"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/rating"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/servicetype"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/settings"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/crypto"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/events"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/gateway"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/jwtx"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/outbox"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/saga"
	"github.com/aeroxe/laundry-pro/backend/internal/server"
	"gorm.io/gorm"
)

// Container bundles every constructed dependency.
type Container struct {
	Config    *config.Config
	DB        *gorm.DB
	Cache     *cache.Cache
	Cipher    *crypto.Cipher
	JWT       *jwtx.Manager
	Outbox    *outbox.Repo
	Publisher events.Publisher
	Registry  *saga.Registry
	Relay     *outbox.Relay
	Sagas     *saga.Store
	Bus       *bus.Bus
	Handlers  *server.Handlers
	rbac      *auth.RbacService
}

// RBACChecker exposes the RBAC service as a permission checker for middleware.
func (c *Container) RBACChecker() *auth.RbacService { return c.rbac }

// Build constructs the container from config and infra.
func Build(cfg *config.Config, db *gorm.DB, cache *cache.Cache, publisher events.Publisher) (*Container, error) {
	cipher, err := crypto.NewCipher(cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}
	jwt := jwtx.NewManager(cfg.JWTSecret, cfg.JWTExpiry)
	auditWriter := audit.NewWriter(db)
	outboxRepo := outbox.NewRepo(db, cfg.OutboxMaxRetry)
	sagaStore := saga.NewStore(db, cache)
	registry := saga.NewRegistry()
	b := bus.New()

	relay := outbox.NewRelay(outboxRepo, registry, publisher, cache, outbox.RelayOptions{
		PollInterval: cfg.OutboxPoll,
		BatchSize:    cfg.OutboxBatch,
		MaxRetries:   cfg.OutboxMaxRetry,
	})

	// RBAC + auth services.
	rbac := auth.NewRbacService(db, cache)
	authSvc := auth.NewAuthService(db, cache, jwt, cipher, cfg, auditWriter)
	userSvc := auth.NewUserService(db, cache, cipher, cfg, auditWriter, rbac)
	roleSvc := auth.NewRoleService(db, auditWriter, rbac)
	logSvc := auth.NewLogService(db)

	// Shop-wide settings double as the currency/tax/prefix source for the
	// money-flow modules (see shopconfig.Provider).
	setSvc := settings.NewService(db, auditWriter)

	// Business modules.
	stSvc := servicetype.NewService(db, auditWriter, setSvc)
	cbSvc := combo.NewService(db, auditWriter, setSvc)
	prSvc := pricing.NewService(stSvc, cbSvc, setSvc, cache)
	// Catalog edits invalidate the pricing cache immediately; the pricing
	// service is wired after combo/servicetype (it consumes them as readers),
	// hence the setters.
	stSvc.SetCatalogInvalidator(prSvc)
	cbSvc.SetCatalogInvalidator(prSvc)
	bkSvc := booking.NewService(db, outboxRepo, b, auditWriter, setSvc)
	pkSvc := pickup.NewService(db, outboxRepo, b, auditWriter)
	odSvc := order.NewService(db, outboxRepo, b, auditWriter, setSvc)
	dvSvc := delivery.NewService(db, outboxRepo, b, auditWriter)
	pySvc := payment.NewService(db, cache, cipher, gateway.NewStub(), auditWriter, b, setSvc)
	gaSvc := garment.NewService(db, auditWriter)
	inSvc := invoice.NewService(db, b, auditWriter, setSvc)
	rtSvc := rating.NewService(db, auditWriter)
	ivSvc := inventory.NewService(db, cache, auditWriter)
	ntSvc := notification.NewService(db, auditWriter)
	anSvc := analytics.NewService(db)

	// Flow is wired after the settings service so saga-issued invoices use the
	// same shop-configured prefix as API-issued ones.
	flow := flows.New(db, outboxRepo, sagaStore, b, setSvc)
	flow.Register(registry)

	handlers := &server.Handlers{
		Auth:         auth.NewHandler(authSvc, userSvc, roleSvc, logSvc),
		ServiceType:  servicetype.NewHandler(stSvc),
		Combo:        combo.NewHandler(cbSvc),
		Pricing:      pricing.NewHandler(prSvc),
		Booking:      booking.NewHandler(bkSvc),
		Pickup:       pickup.NewHandler(pkSvc),
		Order:        order.NewHandler(odSvc),
		Delivery:     delivery.NewHandler(dvSvc),
		Payment:      payment.NewHandler(pySvc),
		Garment:      garment.NewHandler(gaSvc),
		Invoice:      invoice.NewHandler(inSvc),
		Rating:       rating.NewHandler(rtSvc),
		Inventory:    inventory.NewHandler(ivSvc),
		Notification: notification.NewHandler(ntSvc),
		Analytics:    analytics.NewHandler(anSvc),
		Settings:     settings.NewHandler(setSvc),
	}

	return &Container{
		Config:    cfg,
		DB:        db,
		Cache:     cache,
		Cipher:    cipher,
		JWT:       jwt,
		Outbox:    outboxRepo,
		Publisher: publisher,
		Registry:  registry,
		Relay:     relay,
		Sagas:     sagaStore,
		Bus:       b,
		Handlers:  handlers,
		rbac:      rbac,
	}, nil
}

// StartRelay launches the outbox relay worker.
func (c *Container) StartRelay(ctx context.Context) { c.Relay.Start(ctx) }

// StopRelay halts the outbox relay worker.
func (c *Container) StopRelay() { c.Relay.Stop() }
