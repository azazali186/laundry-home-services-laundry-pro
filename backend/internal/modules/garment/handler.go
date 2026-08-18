package garment

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Handler exposes garment endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the garment handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create handles POST /api/v1/garments
//
//	@Summary	Create a garment
//	@Tags		garments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	garment.CreateRequest	true	"Garment"
//	@Success	201		{object}	response.Envelope
//	@Router		/garments [post]
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

// Update handles PATCH /api/v1/garments
//
//	@Summary	Update a garment (id in body)
//	@Tags		garments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	garment.UpdateRequest	true	"Update"
//	@Success	200		{object}	response.Envelope
//	@Router		/garments [patch]
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

// Delete handles DELETE /api/v1/garments
//
//	@Summary	Delete a garment (id in body)
//	@Tags		garments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	garment.IDRequest	true	"Id"
//	@Success	200		{object}	response.Envelope
//	@Router		/garments [delete]
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

// List handles POST /api/v1/garments/list
//
//	@Summary	List garments with offset pagination
//	@Tags		garments
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	garment.ListRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/garments/list [post]
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
