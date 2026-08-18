package pricing

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// Handler exposes the public pricing endpoint.
type Handler struct{ svc *Service }

// NewHandler builds the handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Public handles POST /api/v1/pricing (no auth — landing page catalog).
//
//	@Summary	Public pricing catalog (services, combos, shop defaults)
//	@Tags		pricing
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/pricing [post]
func (h *Handler) Public(ctx context.Context, c *app.RequestContext) {
	view, err := h.svc.Public(ctx)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}
