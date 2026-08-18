package payment

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Handler exposes payment endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the payment handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create handles POST /api/v1/payments
//
//	@Summary	Create a payment
//	@Tags		payments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	payment.CreateRequest	true	"Payment"
//	@Success	201		{object}	response.Envelope
//	@Router		/payments [post]
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

// Capture handles POST /api/v1/payments/capture
//
//	@Summary	Capture a pending payment (idempotent)
//	@Tags		payments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	payment.CaptureRequest	true	"Payment id"
//	@Success	200		{object}	response.Envelope
//	@Router		/payments/capture [post]
func (h *Handler) Capture(ctx context.Context, c *app.RequestContext) {
	var req CaptureRequest
	if err := c.Bind(&req); err != nil || req.ID == uuid.Nil {
		response.Err(c, errors.New(400, "id is required"))
		return
	}
	view, err := h.svc.Capture(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// Refund handles POST /api/v1/payments/refund
//
//	@Summary	Refund a paid payment
//	@Tags		payments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	payment.RefundRequest	true	"Payment id + amount"
//	@Success	200		{object}	response.Envelope
//	@Router		/payments/refund [post]
func (h *Handler) Refund(ctx context.Context, c *app.RequestContext) {
	var req RefundRequest
	if err := c.Bind(&req); err != nil || req.ID == uuid.Nil {
		response.Err(c, errors.New(400, "id is required"))
		return
	}
	view, err := h.svc.Refund(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// Update handles PATCH /api/v1/payments
//
//	@Summary	Update a payment (id in body)
//	@Tags		payments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	payment.UpdateRequest	true	"Update"
//	@Success	200		{object}	response.Envelope
//	@Router		/payments [patch]
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

// Delete handles DELETE /api/v1/payments
//
//	@Summary	Delete a payment (id in body)
//	@Tags		payments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	payment.IDRequest	true	"Id"
//	@Success	200		{object}	response.Envelope
//	@Router		/payments [delete]
func (h *Handler) Delete(ctx context.Context, c *app.RequestContext) {
	var req IDRequest
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

// List handles POST /api/v1/payments/list
//
//	@Summary	List payments with offset pagination
//	@Tags		payments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	payment.ListRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/payments/list [post]
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
