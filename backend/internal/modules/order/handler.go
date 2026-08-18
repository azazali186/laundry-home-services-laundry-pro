package order

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Handler exposes order endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the order handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create handles POST /api/v1/orders
//
//	@Summary	Create an order
//	@Tags		orders
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	order.CreateRequest	true	"Order"
//	@Success	201		{object}	response.Envelope
//	@Router		/orders [post]
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

// Update handles PATCH /api/v1/orders
//
//	@Summary	Update an order (id in body)
//	@Tags		orders
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	order.UpdateRequest	true	"Update"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders [patch]
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

// UpdateStatus handles POST /api/v1/orders/status
//
//	@Summary	Transition an order's status
//	@Tags		orders
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	order.UpdateStatusRequest	true	"Transition"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders/status [post]
func (h *Handler) UpdateStatus(ctx context.Context, c *app.RequestContext) {
	var req UpdateStatusRequest
	if err := c.Bind(&req); err != nil || req.ID == uuid.Nil {
		response.Err(c, errors.New(400, "id and status are required"))
		return
	}
	req.Status = NormalizeStatus(req.Status)
	view, err := h.svc.UpdateStatus(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// Delete handles DELETE /api/v1/orders
//
//	@Summary	Delete an order (id in body)
//	@Tags		orders
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	order.IDRequest	true	"Id"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders [delete]
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

// List handles POST /api/v1/orders/list
//
//	@Summary	List orders with offset pagination
//	@Tags		orders
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	order.ListRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders/list [post]
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
