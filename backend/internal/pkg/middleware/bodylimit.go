package middleware

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// BodyLimit rejects requests whose declared body exceeds maxBytes.
// A hard server-level cap is also configured at engine creation.
func BodyLimit(maxBytes int) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if c.Request.Header.ContentLength() > maxBytes {
			response.AbortErr(c, errors.ErrLimitExceeded)
			return
		}
		c.Next(ctx)
	}
}
