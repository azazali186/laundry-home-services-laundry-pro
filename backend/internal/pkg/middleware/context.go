package middleware

import (
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Context keys stored on the hertz request context.
const (
	ctxUserID    = "user_id"
	ctxEmail     = "email"
	ctxRoles     = "roles"
	ctxRequestID = "request_id"
)

// SetIdentity stores the authenticated identity on the context.
func SetIdentity(c *app.RequestContext, userID uuid.UUID, email string, roles []string) {
	c.Set(ctxUserID, userID)
	c.Set(ctxEmail, email)
	c.Set(ctxRoles, roles)
}

// UserID returns the authenticated user id or an error.
func UserID(c *app.RequestContext) (uuid.UUID, error) {
	v, ok := c.Get(ctxUserID)
	if !ok {
		return uuid.Nil, errors.ErrUnauthorized
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.ErrUnauthorized
	}
	return id, nil
}

// Email returns the authenticated user's email.
func Email(c *app.RequestContext) string {
	if v, ok := c.Get(ctxEmail); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Roles returns the authenticated user's roles.
func Roles(c *app.RequestContext) []string {
	if v, ok := c.Get(ctxRoles); ok {
		if s, ok := v.([]string); ok {
			return s
		}
	}
	return nil
}

// SetRequestID stores the traceable request id.
func SetRequestID(c *app.RequestContext, id string) { c.Set(ctxRequestID, id) }

// RequestID reads the traceable request id.
func RequestID(c *app.RequestContext) string {
	if v, ok := c.Get(ctxRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
