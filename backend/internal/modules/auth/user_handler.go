package auth

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// CreateUser handles POST /api/v1/users
//
//	@Summary	Create a user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.CreateUserRequest	true	"User"
//	@Success	201		{object}	response.Envelope
//	@Router		/users [post]
func (h *Handler) CreateUser(ctx context.Context, c *app.RequestContext) {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	view, err := h.users.Create(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, view)
}

// UpdateUser handles PATCH /api/v1/users
//
//	@Summary	Update a user (id in body)
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.UpdateUserRequest	true	"User update"
//	@Success	200		{object}	response.Envelope
//	@Router		/users [patch]
func (h *Handler) UpdateUser(ctx context.Context, c *app.RequestContext) {
	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil || req.ID == uuidNil() {
		response.Err(c, errors.New(400, "user id is required"))
		return
	}
	view, err := h.users.Update(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// DeleteUser handles DELETE /api/v1/users
//
//	@Summary	Delete a user (id in body)
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.DeleteUserRequest	true	"User id"
//	@Success	200		{object}	response.Envelope
//	@Router		/users [delete]
func (h *Handler) DeleteUser(ctx context.Context, c *app.RequestContext) {
	var req DeleteUserRequest
	if err := c.Bind(&req); err != nil || req.ID == uuidNil() {
		response.Err(c, errors.New(400, "user id is required"))
		return
	}
	if err := h.users.Delete(ctx, req, audit.MetaFrom(c)); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, map[string]bool{"deleted": true})
}

// ListUsers handles POST /api/v1/users/list
//
//	@Summary	List users with offset pagination
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.ListUsersRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/users/list [post]
func (h *Handler) ListUsers(ctx context.Context, c *app.RequestContext) {
	var req ListUsersRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	items, pag, summary, err := h.users.List(ctx, req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Listed(c, items, pag, summary)
}
