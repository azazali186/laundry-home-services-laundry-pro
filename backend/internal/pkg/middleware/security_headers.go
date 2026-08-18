package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// SecurityHeaders adds standard hardening headers to every response.
// In production it sets HSTS with a long max-age; in development it
// only adds the non-HSTS headers so local HTTP still works.
func SecurityHeaders(production bool) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Response.Header.Set("X-Content-Type-Options", "nosniff")
		c.Response.Header.Set("X-Frame-Options", "DENY")
		c.Response.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Response.Header.Set("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=()")
		c.Response.Header.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

		if production {
			c.Response.Header.Set("Strict-Transport-Security",
				"max-age=63072000; includeSubDomains; preload")
		}
		c.Next(ctx)
	}
}
