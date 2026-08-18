// LaundryPro backend entrypoint.
//
//	@title			AeroXe LaundryPro API
//	@version		1.0
//	@description	Laundry service management: bookings, pickups, orders, deliveries, payments, inventory, RBAC, audit & analytics.
//	@host			localhost:9090
//	@BasePath		/api/v1
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT access token: "Bearer <token>"
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/aeroxe/laundry-pro/backend/docs/swagger" // registers OpenAPI docs for http-swagger

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/container"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/db"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/events"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/migrate"
	"github.com/aeroxe/laundry-pro/backend/internal/server"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "run migrations and exit")
	rollbackTo := flag.String("rollback-to", "", "rollback to a specific migration version and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config error: %v", err)
		os.Exit(1)
	}
	logger.Init(logger.Options{Level: cfg.LogLevel, JSON: cfg.LogFormat == "json"})

	gdb, err := db.Open(db.DefaultOptions(cfg.DatabaseURL))
	if err != nil {
		logger.Error("database error: %v", err)
		os.Exit(1)
	}
	if err := migrate.Run(gdb); err != nil {
		logger.Error("migration error: %v", err)
		os.Exit(1)
	}
	if *rollbackTo != "" {
		if err := migrate.Rollback(gdb, *rollbackTo); err != nil {
			logger.Error("rollback error: %v", err)
			os.Exit(1)
		}
		logger.Info("rolled back to %s", *rollbackTo)
		return
	}
	if *migrateOnly {
		logger.Info("migrations applied")
		return
	}

	redis, err := cache.New(cfg.RedisURL, cfg.RedisPrefix)
	if err != nil {
		logger.Error("redis error: %v", err)
		os.Exit(1)
	}

	var publisher events.Publisher = events.NoopPublisher{}
	if cfg.NATSEnabled {
		if p, err := events.NewNATSPublisher(cfg.NATSURL); err == nil {
			publisher = p
			defer p.Close()
		} else {
			logger.Warn("nats unavailable, degrading to noop publisher: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := container.Build(cfg, gdb, redis, publisher)
	if err != nil {
		logger.Error("container build error: %v", err)
		os.Exit(1)
	}

	// Bootstrap roles + admin user, then the default pricing catalog (service
	// types + combos), then seed permissions from routes.
	if err := server.SeedBootstrap(ctx, gdb, cfg, app.Cipher); err != nil {
		logger.Error("seed error: %v", err)
		os.Exit(1)
	}
	if err := server.SeedCatalog(ctx, gdb); err != nil {
		logger.Error("catalog seed error: %v", err)
		os.Exit(1)
	}

	engine := server.NewEngine(server.Deps{
		Config:   cfg,
		DB:       gdb,
		Cache:    redis,
		JWT:      app.JWT,
		Checker:  app.RBACChecker(),
		Bus:      app.Bus,
		Handlers: app.Handlers,
	})

	if err := server.SeedPermissions(ctx, gdb, engine); err != nil {
		logger.Error("permission seed error: %v", err)
		os.Exit(1)
	}

	app.StartRelay(ctx)
	defer app.StopRelay()

	// Bridge Hertz's graceful shutdown to our signal context. Spin() drains
	// in-flight requests up to SHUTDOWN_TIMEOUT before returning; a run error
	// (e.g. port in use) exits non-zero.
	var runErr error
	engine.SetCustomSignalWaiter(func(errCh chan error) error {
		select {
		case <-ctx.Done():
			return nil // signal received: graceful shutdown
		case err := <-errCh:
			runErr = err
			return err // server failed to start/run: close and exit
		}
	})

	logger.Info("LaundryPro backend listening on :%s (env=%s)", cfg.Port, cfg.Env)
	engine.Spin()
	if runErr != nil {
		logger.Error("server error: %v", runErr)
		os.Exit(1)
	}
	logger.Info("graceful shutdown complete")
}
