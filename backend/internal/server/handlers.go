package server

import (
	"github.com/aeroxe/laundry-pro/backend/internal/modules/analytics"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/auth"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/booking"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/combo"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/delivery"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/garment"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/inventory"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/invoice"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/notification"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/order"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/payment"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/pickup"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/pricing"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/rating"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/servicetype"
	"github.com/aeroxe/laundry-pro/backend/internal/modules/settings"
)

// Handlers aggregates every module handler for route registration.
type Handlers struct {
	Auth         *auth.Handler
	ServiceType  *servicetype.Handler
	Combo        *combo.Handler
	Pricing      *pricing.Handler
	Booking      *booking.Handler
	Pickup       *pickup.Handler
	Order        *order.Handler
	Delivery     *delivery.Handler
	Payment      *payment.Handler
	Garment      *garment.Handler
	Invoice      *invoice.Handler
	Rating       *rating.Handler
	Inventory    *inventory.Handler
	Notification *notification.Handler
	Analytics    *analytics.Handler
	Settings     *settings.Handler
}
