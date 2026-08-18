package notification

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Handler exposes notification endpoints.
type Handler struct{ svc *Service }

// NewHandler builds the notification handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create handles POST /api/v1/notifications
//
//	@Summary	Create a notification
//	@Tags		notifications
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	notification.CreateRequest	true	"Notification"
//	@Success	201		{object}	response.Envelope
//	@Router		/notifications [post]
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

// MarkRead handles POST /api/v1/notifications/mark-read
//
//	@Summary	Mark a notification read
//	@Tags		notifications
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	notification.MarkReadRequest	true	"Notification id"
//	@Success	200		{object}	response.Envelope
//	@Router		/notifications/mark-read [post]
func (h *Handler) MarkRead(ctx context.Context, c *app.RequestContext) {
	var req MarkReadRequest
	if err := c.Bind(&req); err != nil || req.ID == uuid.Nil {
		response.Err(c, errors.New(400, "id is required"))
		return
	}
	view, err := h.svc.MarkRead(ctx, req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// List handles POST /api/v1/notifications/list
//
//	@Summary	List notifications with offset pagination
//	@Tags		notifications
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	notification.ListRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/notifications/list [post]
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
