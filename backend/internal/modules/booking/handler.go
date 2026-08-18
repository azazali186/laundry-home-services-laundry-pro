package booking

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Handler exposes booking endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the booking handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create handles POST /api/v1/bookings
//
//	@Summary	Create a booking
//	@Tags		bookings
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	booking.CreateRequest	true	"Booking"
//	@Success	201		{object}	response.Envelope
//	@Router		/bookings [post]
func (h *Handler) Create(ctx context.Context, c *app.RequestContext) {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	view, err := h.svc.Create(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, view)
}

// Update handles PATCH /api/v1/bookings
//
//	@Summary	Update a booking (id in body)
//	@Tags		bookings
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	booking.UpdateRequest	true	"Update"
//	@Success	200		{object}	response.Envelope
//	@Router		/bookings [patch]
func (h *Handler) Update(ctx context.Context, c *app.RequestContext) {
	var req UpdateRequest
	if err := c.Bind(&req); err != nil || req.ID == uuid.Nil {
		response.Err(c, errors.New(400, "id is required"))
		return
	}
	view, err := h.svc.Update(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// Confirm handles POST /api/v1/bookings/confirm
//
//	@Summary	Confirm a pending booking
//	@Tags		bookings
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	booking.CancelRequest	true	"Booking id"
//	@Success	200		{object}	response.Envelope
//	@Router		/bookings/confirm [post]
func (h *Handler) Confirm(ctx context.Context, c *app.RequestContext) {
	var req CancelRequest
	if err := c.Bind(&req); err != nil || req.ID == uuid.Nil {
		response.Err(c, errors.New(400, "id is required"))
		return
	}
	view, err := h.svc.Confirm(ctx, req.ID, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// Cancel handles POST /api/v1/bookings/cancel
//
//	@Summary	Cancel a booking
//	@Tags		bookings
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	booking.CancelRequest	true	"Booking id + reason"
//	@Success	200		{object}	response.Envelope
//	@Router		/bookings/cancel [post]
func (h *Handler) Cancel(ctx context.Context, c *app.RequestContext) {
	var req CancelRequest
	if err := c.Bind(&req); err != nil || req.ID == uuid.Nil {
		response.Err(c, errors.New(400, "id is required"))
		return
	}
	view, err := h.svc.Cancel(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// Delete handles DELETE /api/v1/bookings
//
//	@Summary	Delete a booking (id in body)
//	@Tags		bookings
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	booking.DeleteRequest	true	"Id"
//	@Success	200		{object}	response.Envelope
//	@Router		/bookings [delete]
func (h *Handler) Delete(ctx context.Context, c *app.RequestContext) {
	var req DeleteRequest
	if err := c.Bind(&req); err != nil || req.ID == uuid.Nil {
		response.Err(c, errors.New(400, "id is required"))
		return
	}
	if err := h.svc.Delete(ctx, req, audit.MetaFrom(c)); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, map[string]bool{"deleted": true})
}

// List handles POST /api/v1/bookings/list
//
//	@Summary	List bookings with offset pagination
//	@Tags		bookings
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	booking.ListRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/bookings/list [post]
func (h *Handler) List(ctx context.Context, c *app.RequestContext) {
	var req ListRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	items, pag, summary, err := h.svc.List(ctx, req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Listed(c, items, pag, summary)
}
