package middleware

import (
	"context"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/jwtx"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// sessionTTL is the lifetime of a server-side session.
const sessionTTL = 24 * time.Hour

// renewThreshold: when remaining TTL drops below this the session is extended.
const renewThreshold = 30 * time.Minute

// Auth validates the Bearer token and enforces single-sign-on semantics:
// the token fingerprint must match the active session stored in Redis.
func Auth(cache *cache.Cache, manager *jwtx.Manager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authorization := string(c.Request.Header.Get("Authorization"))
		if len(authorization) <= 7 {
			response.AbortErr(c, errors.ErrInvalidToken)
			return
		}
		tokenStr := authorization[7:]
		claims, err := manager.Parse(tokenStr)
		if err != nil {
			logger.Warn("token parse failed: %v", err)
			response.AbortErr(c, errors.ErrInvalidToken)
			return
		}

		sessionKey := "session:" + claims.UserID.String()
		stored, ttl, err := cache.GetWithTTL(ctx, sessionKey)
		if err != nil {
			response.AbortErr(c, errors.ErrTokenExpired)
			return
		}
		fingerprint := jwtx.SessionFingerprint(tokenStr, claims.UserID)
		if fingerprint != stored {
			response.AbortErr(c, errors.ErrTokenExpired)
			return
		}
		// Sliding renewal: extend sessions that are about to expire.
		if ttl < renewThreshold {
			if err := cache.Set(ctx, sessionKey, fingerprint, sessionTTL); err != nil {
				response.AbortErr(c, errors.ErrTokenExpired)
				return
			}
		}

		SetIdentity(c, claims.UserID, claims.Email, claims.Roles)
		c.Next(ctx)
	}
}
