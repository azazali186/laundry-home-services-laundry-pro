package server

import (
	"context"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

// pingable abstracts anything that can report connectivity.
type pingable interface {
	Ping(ctx context.Context) error
}

// HealthDeps supplies connectivity probes for the readiness check.
type HealthDeps struct {
	DB    *gorm.DB
	Cache pingable
}

// HealthHandler reports liveness: the process is up. It never depends on
// external services, so orchestrators can restart a wedged instance.
func HealthHandler(cfg *config.Config) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		response.OK(c, map[string]any{
			"status": "ok",
			"env":    cfg.Env,
			"time":   nowUTC(),
		})
	}
}

// ReadyHandler reports readiness by probing Postgres and Redis with a
// bounded timeout. Returns 503 with details when a dependency is down so
// load balancers stop routing traffic to the instance.
func ReadyHandler(deps HealthDeps, cfg *config.Config) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		checks := map[string]string{}
		ready := true

		if deps.DB != nil {
			sqlDB, err := deps.DB.DB()
			if err != nil {
				ready = false
				checks["database"] = "unavailable: " + err.Error()
			} else {
				pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
				err := sqlDB.PingContext(pctx)
				cancel()
				if err != nil {
					ready = false
					checks["database"] = "unreachable"
				} else {
					checks["database"] = "ok"
				}
			}
		}
		if deps.Cache != nil {
			pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := deps.Cache.Ping(pctx)
			cancel()
			if err != nil {
				ready = false
				checks["cache"] = "unreachable"
			} else {
				checks["cache"] = "ok"
			}
		}

		body := map[string]any{"checks": checks, "env": cfg.Env, "time": nowUTC()}
		if ready {
			response.OK(c, body)
			return
		}
		response.Write(c, 503, 503, "not ready", body, nil, nil)
	}
}

func nowUTC() string {
	return timeNow().UTC().Format("2006-01-02T15:04:05Z07:00")
}
