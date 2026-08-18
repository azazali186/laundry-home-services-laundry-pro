package middleware

import (
	"context"
	"net/http"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/cloudwego/hertz/pkg/app"
)

// Recovery converts panics into 500 responses without crashing the server.
func Recovery() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered: %v", recovered)
				c.JSON(http.StatusInternalServerError, map[string]any{
					"code":    http.StatusInternalServerError,
					"message": "internal server error",
				})
				c.Abort()
			}
		}()
		c.Next(ctx)
	}
}
