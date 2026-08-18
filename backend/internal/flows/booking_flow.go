// Package flows wires domain events to saga handlers, implementing the
// BookingFlow saga: booking -> pickup -> order -> delivery, with
// compensation (booking cancellation) when a step fails.
package flows

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/bus"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/modkit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/outbox"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/saga"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/shopconfig"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Flow wires saga handlers onto the registry.
type Flow struct {
	db     *gorm.DB
	outbox *outbox.Repo
	sagas  *saga.Store
	bus    *bus.Bus
	shop   shopconfig.Provider
	now    func() time.Time
}

// New builds the booking flow. shop supplies the shop-wide defaults (invoice
// prefix) so saga-issued invoices use the same numbering as API-issued ones.
func New(db *gorm.DB, outbox *outbox.Repo, sagas *saga.Store, bus *bus.Bus, shop shopconfig.Provider) *Flow {
	shop = shopconfig.Guard(shop)
	return &Flow{db: db, outbox: outbox, sagas: sagas, bus: bus, shop: shop, now: time.Now}
}

// Register binds all flow handlers.
func (f *Flow) Register(registry *saga.Registry) {
	registry.Register("booking_created", f.onBookingCreated)
	registry.Register("pickup_completed", f.onPickupCompleted)
	registry.Register("order_status_changed", f.onOrderStatusChanged)
	registry.Register("delivery_delivered", f.onDeliveryDelivered)
}

// bookingCreatedPayload mirrors the outbox event payload.
type bookingCreatedPayload struct {
	BookingID     string     `json:"booking_id"`
	BookingNo     string     `json:"booking_no"`
	CustomerID    string     `json:"customer_id"`
	ServiceTypeID string     `json:"service_type_id"`
	PickupAddress string     `json:"pickup_address"`
	PickupFrom    *time.Time `json:"pickup_from"`
	PickupTo      *time.Time `json:"pickup_to"`
}

// onBookingCreated starts the saga and schedules a pickup. On failure it
// compensates by cancelling the booking.
func (f *Flow) onBookingCreated(ctx context.Context, ev saga.Event) error {
	var p bookingCreatedPayload
	if err := ev.Decode(&p); err != nil {
		return err
	}
	bookingID, err := parseUUID(p.BookingID)
	if err != nil {
		return err
	}
	sagaID, err := f.sagas.Start(ctx, "booking_flow", p.BookingID, map[string]any{
		"booking_id": p.BookingID, "customer_id": p.CustomerID,
		"service_type_id": p.ServiceTypeID, "pickup_address": p.PickupAddress,
	})
	if err != nil {
		return err
	}

	pickupAt := time.Now()
	if p.PickupFrom != nil {
		pickupAt = *p.PickupFrom
	}
	pickup := models.Pickup{
		BookingID:   bookingID,
		ScheduledAt: pickupAt,
		Address:     p.PickupAddress,
		Status:      models.PickupScheduled,
	}
	err = f.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pickup).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Booking{}).Where("id = ?", p.BookingID).
			Update("status", models.BookingConfirmed).Error; err != nil {
			return err
		}
		return f.outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "pickup", AggregateID: pickup.ID.String(),
			EventType: "pickup_scheduled",
			Payload:   map[string]any{"pickup_id": pickup.ID.String(), "booking_id": p.BookingID},
		})
	})
	if err != nil {
		// Compensation: cancel the booking, mark the saga compensated.
		f.db.WithContext(ctx).Model(&models.Booking{}).Where("id = ?", p.BookingID).
			Update("status", models.BookingCancelled)
		_ = f.sagas.Compensate(ctx, sagaID, err)
		return err
	}
	_ = f.sagas.Advance(ctx, sagaID, "pickup_scheduled", map[string]any{"pickup_id": pickup.ID.String()})
	f.bus.Publish(bus.Event{Type: "pickup_scheduled", AggregateID: pickup.ID.String(), Data: pickup})
	return nil
}

// onPickupCompleted creates the order for the booking.
func (f *Flow) onPickupCompleted(ctx context.Context, ev saga.Event) error {
	var p struct {
		PickupID  string `json:"pickup_id"`
		BookingID string `json:"booking_id"`
	}
	if err := ev.Decode(&p); err != nil {
		return err
	}
	var booking models.Booking
	if err := f.db.WithContext(ctx).First(&booking, "id = ?", p.BookingID).Error; err != nil {
		return err
	}
	order := models.Order{
		OrderNo:       modkit.NextNo("ORD"),
		BookingID:     booking.ID,
		CustomerID:    booking.CustomerID,
		ServiceTypeID: booking.ServiceTypeID,
		Status:        models.OrderCreated,
		Priority:      "normal",
		Currency:      booking.Currency,
		GrandTotal:    booking.EstimatedAmount,
		TotalAmount:   booking.EstimatedAmount,
	}
	err := f.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return f.outbox.Enqueue(ctx, tx, outbox.Event{
			AggregateType: "order", AggregateID: order.ID.String(),
			EventType: "order_created",
			Payload:   map[string]any{"order_id": order.ID.String(), "booking_id": booking.ID.String()},
		})
	})
	if err != nil {
		return err
	}
	if sag := f.sagaForBooking(ctx, booking.ID.String()); sag != nil {
		_ = f.sagas.Advance(ctx, sag.SagaID, "order_created", map[string]any{"order_id": order.ID.String(), "pickup_id": p.PickupID})
	}
	f.bus.Publish(bus.Event{Type: "order_created", AggregateID: order.ID.String(), Data: order})
	logger.Info("booking flow: order %s created for booking %s", order.ID, booking.ID)
	return nil
}

// onOrderStatusChanged creates a delivery when an order becomes ready.
func (f *Flow) onOrderStatusChanged(ctx context.Context, ev saga.Event) error {
	var p struct {
		OrderID string `json:"order_id"`
		To      string `json:"to"`
	}
	if err := ev.Decode(&p); err != nil {
		return err
	}
	if p.To != models.OrderReady {
		return nil
	}
	var order models.Order
	if err := f.db.WithContext(ctx).First(&order, "id = ?", p.OrderID).Error; err != nil {
		return err
	}
	delivery := models.Delivery{
		OrderID:     order.ID,
		ScheduledAt: time.Now().Add(24 * time.Hour),
		Status:      models.DeliveryScheduled,
	}
	if err := f.db.WithContext(ctx).Create(&delivery).Error; err != nil {
		return err
	}
	if sag := f.sagaForBooking(ctx, order.BookingID.String()); sag != nil {
		_ = f.sagas.Advance(ctx, sag.SagaID, "delivery_scheduled", map[string]any{"delivery_id": delivery.ID.String(), "order_id": order.ID.String()})
	}
	f.bus.Publish(bus.Event{Type: "delivery_scheduled", AggregateID: delivery.ID.String(), Data: delivery})
	logger.Info("order %s ready: delivery %s scheduled", order.ID, delivery.ID)
	return nil
}

// onDeliveryDelivered finalizes the order, issues the invoice and notifies.
func (f *Flow) onDeliveryDelivered(ctx context.Context, ev saga.Event) error {
	var p struct {
		DeliveryID string `json:"delivery_id"`
		OrderID    string `json:"order_id"`
	}
	if err := ev.Decode(&p); err != nil {
		return err
	}
	var order models.Order
	if err := f.db.WithContext(ctx).First(&order, "id = ?", p.OrderID).Error; err != nil {
		return err
	}
	// Use the shop-configured invoice prefix (fallback INV) so saga-issued
	// invoices share numbering with API-issued ones; settings read failures
	// degrade to the fallback rather than blocking delivery finalization.
	prefix := "INV"
	if d, err := f.shop.Defaults(ctx); err == nil {
		if p := strings.TrimSpace(d.InvoicePrefix); p != "" {
			prefix = p
		}
	}
	invoice := models.Invoice{
		InvoiceNo: modkit.NextNo(prefix),
		OrderID:   order.ID,
		IssueDate: time.Now(),
		Subtotal:  order.TotalAmount,
		TaxAmount: order.TaxAmount,
		Discount:  order.DiscountAmount,
		Total:     order.GrandTotal,
		Status:    models.InvoiceIssued,
	}
	err := f.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&order).Update("status", models.OrderDelivered).Error; err != nil {
			return err
		}
		return tx.Create(&invoice).Error
	})
	if err != nil {
		return err
	}
	if sag := f.sagaForBooking(ctx, order.BookingID.String()); sag != nil {
		_ = f.sagas.Complete(ctx, sag.SagaID, map[string]any{"invoice_id": invoice.ID.String(), "order_id": order.ID.String()})
	}
	f.bus.Publish(bus.Event{Type: "order_delivered", AggregateID: order.ID.String(), Data: order})
	logger.Info("order %s delivered; invoice issued", order.ID)
	return nil
}

// sagaForBooking resolves the running saga for a booking (nil if none).
func (f *Flow) sagaForBooking(ctx context.Context, bookingID string) *models.SagaState {
	sag, err := f.sagas.FindByAggregate(ctx, bookingID)
	if err != nil {
		logger.Warn("saga lookup for booking %s failed: %v", bookingID, err)
		return nil
	}
	return sag
}

// parseUUID validates an event payload id. Returning an error (instead of
// silently producing uuid.Nil) makes the event fail and retry rather than
// writing orphaned rows with a nil foreign key.
func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid uuid %q in event payload: %w", s, err)
	}
	return id, nil
}
