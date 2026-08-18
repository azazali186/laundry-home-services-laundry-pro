package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// window is the rate limit sliding window length.
const window = time.Minute

// RouteLimit maps a path prefix to a per-minute limit for that route.
// More restrictive limits win; unmatched routes fall back to the global limit.
type RouteLimit struct {
	Prefix string
	Limit  int64
}

// RateLimit enforces a per-identity limit using a Redis counter. When
// routeLimits is provided, the request path is matched against each prefix
// (longest prefix first) and its limit is used instead of the global one,
// giving per-route protection for hot or dangerous endpoints.
func RateLimit(cache *cache.Cache, limit int64, routeLimits ...RouteLimit) app.HandlerFunc {
	return rateLimit(cache, "api", limit, routeLimits)
}

// RateLimitScoped is RateLimit with a distinct counter namespace. Use it for
// limiters outside the API group (e.g. public endpoints) so their per-IP
// budget never shares a counter with the protected group's limiter.
func RateLimitScoped(cache *cache.Cache, scope string, limit int64, routeLimits ...RouteLimit) app.HandlerFunc {
	return rateLimit(cache, scope, limit, routeLimits)
}

func rateLimit(cache *cache.Cache, scope string, limit int64, routeLimits []RouteLimit) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		key := "ratelimit:" + scope + ":" + c.ClientIP()
		if id, err := UserID(c); err == nil {
			key = "ratelimit:" + scope + ":user:" + id.String()
		}
		path := string(c.Path())
		rl := limitFor(path, routeLimits, limit)
		count, err := cache.Incr(ctx, key, window)
		if err != nil {
			// Degrade open on cache failure; never block traffic on infra errors.
			c.Next(ctx)
			return
		}
		if count > rl {
			response.AbortErr(c, errors.ErrTooMany)
			return
		}
		c.Next(ctx)
	}
}

// limitFor picks the longest matching route prefix, else the global limit.
// A route ending in "/list" (any module) is always capped at the global
// limit, keeping read-heavy list endpoints off the tightened write caps.
func limitFor(path string, routes []RouteLimit, global int64) int64 {
	if strings.HasSuffix(path, "/list") {
		return global
	}
	best := -1
	bestLen := 0
	for i, r := range routes {
		if strings.HasPrefix(path, r.Prefix) && len(r.Prefix) > bestLen {
			best = i
			bestLen = len(r.Prefix)
		}
	}
	if best >= 0 {
		return routes[best].Limit
	}
	return global
}
