package middleware

import (
	"context"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/cloudwego/hertz/pkg/app"
)

// RequestLog logs one line per request with status and latency.
func RequestLog() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		c.Next(ctx)
		logger.Info("%s %s -> %d (%s) ip=%s rid=%s",
			string(c.Method()), string(c.Path()), c.Response.StatusCode(),
			time.Since(start).Round(time.Microsecond), c.ClientIP(), RequestID(c))
	}
}
