package analytics

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// Handler exposes analytics endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the analytics handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Dashboard handles POST /api/v1/analytics/dashboard
//
//	@Summary	Operational dashboard aggregates
//	@Tags		analytics
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	analytics.DashboardRequest	true	"Date range"
//	@Success	200		{object}	response.Envelope
//	@Router		/analytics/dashboard [post]
func (h *Handler) Dashboard(ctx context.Context, c *app.RequestContext) {
	var req DashboardRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	result, err := h.svc.Dashboard(ctx, req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, result)
}
