package server

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	httpSwagger "github.com/swaggo/http-swagger"
)

// SwaggerHandler serves the generated Swagger UI at /swagger/*any.
// github.com/swaggo/http-swagger is net/http based; we bridge it onto the
// Hertz engine with serveNetHTTP.
func SwaggerHandler(ctx context.Context, c *app.RequestContext) {
	handler := httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
	)
	serveNetHTTP(c, handler)
}
