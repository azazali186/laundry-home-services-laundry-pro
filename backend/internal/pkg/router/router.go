// Package router declares the registration surface for module routes.
// Hertz's *route.RouterGroup is used directly so module wiring stays
// framework-native and testable with real route groups.
package router

import "github.com/cloudwego/hertz/pkg/route"

// Registrar is satisfied by *route.RouterGroup (and *server.Hertz via Group).
type Registrar = *route.RouterGroup
