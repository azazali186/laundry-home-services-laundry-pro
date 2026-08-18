package servicetype

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Handler exposes service type endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create handles POST /api/v1/service-types
//
//	@Summary	Create a service type
//	@Tags		service-types
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	servicetype.CreateRequest	true	"Service type"
//	@Success	201		{object}	response.Envelope
//	@Router		/service-types [post]
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

// Update handles PATCH /api/v1/service-types
//
//	@Summary	Update a service type (id in body)
//	@Tags		service-types
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	servicetype.UpdateRequest	true	"Update"
//	@Success	200		{object}	response.Envelope
//	@Router		/service-types [patch]
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

// Delete handles DELETE /api/v1/service-types
//
//	@Summary	Delete a service type (id in body)
//	@Tags		service-types
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	servicetype.DeleteRequest	true	"Id"
//	@Success	200		{object}	response.Envelope
//	@Router		/service-types [delete]
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

// List handles POST /api/v1/service-types/list
//
//	@Summary	List service types with offset pagination
//	@Tags		service-types
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	servicetype.ListRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/service-types/list [post]
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
