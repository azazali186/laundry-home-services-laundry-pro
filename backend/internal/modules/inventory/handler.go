package inventory

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Handler exposes inventory endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the inventory handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create handles POST /api/v1/inventory
//
//	@Summary	Create an inventory item
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	inventory.CreateRequest	true	"Item"
//	@Success	201		{object}	response.Envelope
//	@Router		/inventory [post]
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

// Update handles PATCH /api/v1/inventory
//
//	@Summary	Update an inventory item (id in body)
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	inventory.UpdateRequest	true	"Update"
//	@Success	200		{object}	response.Envelope
//	@Router		/inventory [patch]
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

// Adjust handles POST /api/v1/inventory/adjust
//
//	@Summary	Adjust stock by a signed delta (locked)
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	inventory.AdjustRequest	true	"Delta"
//	@Success	200		{object}	response.Envelope
//	@Router		/inventory/adjust [post]
func (h *Handler) Adjust(ctx context.Context, c *app.RequestContext) {
	var req AdjustRequest
	if err := c.Bind(&req); err != nil || req.ID == uuid.Nil {
		response.Err(c, errors.New(400, "id is required"))
		return
	}
	view, err := h.svc.AdjustStock(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// Delete handles DELETE /api/v1/inventory
//
//	@Summary	Delete an inventory item (id in body)
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	inventory.IDRequest	true	"Id"
//	@Success	200		{object}	response.Envelope
//	@Router		/inventory [delete]
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

// List handles POST /api/v1/inventory/list
//
//	@Summary	List inventory with offset pagination
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	inventory.ListRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/inventory/list [post]
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
