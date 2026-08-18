package server

import (
	"github.com/aeroxe/laundry-pro/backend/internal/modules/auth"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/router"
)

// RegisterRoutes mounts every module's endpoints on the protected /api/v1
// group. All paths are relative to the group prefix.
func RegisterRoutes(rg router.Registrar, h *Handlers) {
	auth.Register(rg, h.Auth)

	rg.POST("/service-types", h.ServiceType.Create)
	rg.PATCH("/service-types", h.ServiceType.Update)
	rg.DELETE("/service-types", h.ServiceType.Delete)
	rg.POST("/service-types/list", h.ServiceType.List)

	rg.POST("/combos", h.Combo.Create)
	rg.PATCH("/combos", h.Combo.Update)
	rg.DELETE("/combos", h.Combo.Delete)
	rg.POST("/combos/list", h.Combo.List)

	rg.POST("/bookings", h.Booking.Create)
	rg.PATCH("/bookings", h.Booking.Update)
	rg.POST("/bookings/confirm", h.Booking.Confirm)
	rg.POST("/bookings/cancel", h.Booking.Cancel)
	rg.DELETE("/bookings", h.Booking.Delete)
	rg.POST("/bookings/list", h.Booking.List)

	rg.POST("/pickups", h.Pickup.Create)
	rg.PATCH("/pickups", h.Pickup.Update)
	rg.POST("/pickups/complete", h.Pickup.Complete)
	rg.DELETE("/pickups", h.Pickup.Delete)
	rg.POST("/pickups/list", h.Pickup.List)

	rg.POST("/orders", h.Order.Create)
	rg.PATCH("/orders", h.Order.Update)
	rg.POST("/orders/status", h.Order.UpdateStatus)
	rg.DELETE("/orders", h.Order.Delete)
	rg.POST("/orders/list", h.Order.List)

	rg.POST("/deliveries", h.Delivery.Create)
	rg.PATCH("/deliveries", h.Delivery.Update)
	rg.POST("/deliveries/complete", h.Delivery.Complete)
	rg.DELETE("/deliveries", h.Delivery.Delete)
	rg.POST("/deliveries/list", h.Delivery.List)

	rg.POST("/payments", h.Payment.Create)
	rg.POST("/payments/capture", h.Payment.Capture)
	rg.POST("/payments/refund", h.Payment.Refund)
	rg.PATCH("/payments", h.Payment.Update)
	rg.DELETE("/payments", h.Payment.Delete)
	rg.POST("/payments/list", h.Payment.List)

	rg.POST("/garments", h.Garment.Create)
	rg.PATCH("/garments", h.Garment.Update)
	rg.DELETE("/garments", h.Garment.Delete)
	rg.POST("/garments/list", h.Garment.List)

	rg.POST("/invoices", h.Invoice.Create)
	rg.PATCH("/invoices", h.Invoice.Update)
	rg.POST("/invoices/mark-paid", h.Invoice.MarkPaid)
	rg.DELETE("/invoices", h.Invoice.Delete)
	rg.POST("/invoices/list", h.Invoice.List)

	rg.POST("/ratings", h.Rating.Create)
	rg.PATCH("/ratings", h.Rating.Update)
	rg.DELETE("/ratings", h.Rating.Delete)
	rg.POST("/ratings/list", h.Rating.List)

	rg.POST("/inventory", h.Inventory.Create)
	rg.PATCH("/inventory", h.Inventory.Update)
	rg.POST("/inventory/adjust", h.Inventory.Adjust)
	rg.DELETE("/inventory", h.Inventory.Delete)
	rg.POST("/inventory/list", h.Inventory.List)

	rg.POST("/notifications", h.Notification.Create)
	rg.POST("/notifications/mark-read", h.Notification.MarkRead)
	rg.POST("/notifications/list", h.Notification.List)

	rg.POST("/analytics/dashboard", h.Analytics.Dashboard)

	rg.POST("/settings/view", h.Settings.View)
	rg.PATCH("/settings", h.Settings.Update)
}
