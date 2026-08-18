package middleware

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// PermissionChecker resolves whether a user may access a route.
type PermissionChecker interface {
	Check(ctx context.Context, userID uuid.UUID, method, path string) bool
}

// RBAC guards routes using the registered permission checker.
func RBAC(checker PermissionChecker) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		userID, err := UserID(c)
		if err != nil {
			response.AbortErr(c, err)
			return
		}
		if checker.Check(ctx, userID, string(c.Method()), string(c.Path())) {
			c.Next(ctx)
			return
		}
		response.AbortErr(c, errors.ErrForbidden)
	}
}
