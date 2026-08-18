package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// CORS adds permissive CORS headers in development and configurable
// origins in production.
func CORS(allowedOrigins string) app.HandlerFunc {
	origins := map[string]bool{}
	for _, o := range strings.Split(allowedOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins[o] = true
		}
	}
	return func(ctx context.Context, c *app.RequestContext) {
		origin := string(c.Request.Header.Get("Origin"))
		if origin != "" {
			allow := origin
			if !origins["*"] && !origins[origin] {
				allow = ""
			}
			if allow != "" {
				c.Response.Header.Set("Access-Control-Allow-Origin", allow)
				c.Response.Header.Set("Access-Control-Allow-Methods", "POST, PATCH, DELETE, OPTIONS")
				c.Response.Header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id")
				c.Response.Header.Set("Access-Control-Max-Age", "86400")
			}
		}
		if string(c.Method()) == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next(ctx)
	}
}
