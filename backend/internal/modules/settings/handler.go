package settings

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// Handler exposes shop configuration endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the settings handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// View handles POST /api/v1/settings/view
//
//	@Summary	Get the shop configuration
//	@Tags		settings
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.Envelope
//	@Router		/settings/view [post]
func (h *Handler) View(ctx context.Context, c *app.RequestContext) {
	view, err := h.svc.Get(ctx)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// Update handles PATCH /api/v1/settings
//
//	@Summary	Update the shop configuration (singleton, no id needed)
//	@Tags		settings
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	settings.UpdateRequest	true	"Shop settings"
//	@Success	200		{object}	response.Envelope
//	@Router		/settings [patch]
func (h *Handler) Update(ctx context.Context, c *app.RequestContext) {
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	view, err := h.svc.Update(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}
