package auth

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// ListLoginLogs handles POST /api/v1/login-logs/list
//
//	@Summary	List login attempts
//	@Tags		logs
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.ListLoginLogsRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/login-logs/list [post]
func (h *Handler) ListLoginLogs(ctx context.Context, c *app.RequestContext) {
	var req ListLoginLogsRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	items, pag, err := h.logs.ListLoginLogs(ctx, req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Listed(c, items, pag, nil)
}

// ListAuditLogs handles POST /api/v1/audit-logs/list
//
//	@Summary	List audit trail entries
//	@Tags		logs
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.ListAuditLogsRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/audit-logs/list [post]
func (h *Handler) ListAuditLogs(ctx context.Context, c *app.RequestContext) {
	var req ListAuditLogsRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	items, pag, err := h.logs.ListAuditLogs(ctx, req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Listed(c, items, pag, nil)
}
