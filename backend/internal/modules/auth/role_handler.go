package auth

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// CreateRole handles POST /api/v1/roles
//
//	@Summary	Create a role
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.CreateRoleRequest	true	"Role"
//	@Success	201		{object}	response.Envelope
//	@Router		/roles [post]
func (h *Handler) CreateRole(ctx context.Context, c *app.RequestContext) {
	var req CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	view, err := h.roles.Create(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, view)
}

// UpdateRole handles PATCH /api/v1/roles
//
//	@Summary	Update a role (id in body)
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.UpdateRoleRequest	true	"Role update"
//	@Success	200		{object}	response.Envelope
//	@Router		/roles [patch]
func (h *Handler) UpdateRole(ctx context.Context, c *app.RequestContext) {
	var req UpdateRoleRequest
	if err := c.Bind(&req); err != nil || req.ID == uuidNil() {
		response.Err(c, errors.New(400, "role id is required"))
		return
	}
	view, err := h.roles.Update(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// DeleteRole handles DELETE /api/v1/roles
//
//	@Summary	Delete a role (id in body)
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.DeleteRoleRequest	true	"Role id"
//	@Success	200		{object}	response.Envelope
//	@Router		/roles [delete]
func (h *Handler) DeleteRole(ctx context.Context, c *app.RequestContext) {
	var req DeleteRoleRequest
	if err := c.Bind(&req); err != nil || req.ID == uuidNil() {
		response.Err(c, errors.New(400, "role id is required"))
		return
	}
	if err := h.roles.Delete(ctx, req, audit.MetaFrom(c)); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, map[string]bool{"deleted": true})
}

// AssignPermissions handles POST /api/v1/roles/assign-permissions
//
//	@Summary	Replace a role's permission set
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.AssignPermissionsRequest	true	"Assignment"
//	@Success	200		{object}	response.Envelope
//	@Router		/roles/assign-permissions [post]
func (h *Handler) AssignPermissions(ctx context.Context, c *app.RequestContext) {
	var req AssignPermissionsRequest
	if err := c.Bind(&req); err != nil || req.RoleID == uuidNil() {
		response.Err(c, errors.New(400, "role id is required"))
		return
	}
	view, err := h.roles.AssignPermissions(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// ListRoles handles POST /api/v1/roles/list
//
//	@Summary	List roles with offset pagination
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.ListRolesRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/roles/list [post]
func (h *Handler) ListRoles(ctx context.Context, c *app.RequestContext) {
	var req ListRolesRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	items, pag, err := h.roles.List(ctx, req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Listed(c, items, pag, nil)
}

// ListPermissions handles POST /api/v1/permissions/list
//
//	@Summary	List permissions with offset pagination
//	@Tags		permissions
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.ListPermissionsRequest	true	"Page request"
//	@Success	200		{object}	response.Envelope
//	@Router		/permissions/list [post]
func (h *Handler) ListPermissions(ctx context.Context, c *app.RequestContext) {
	var req ListPermissionsRequest
	if err := c.Bind(&req); err != nil {
		response.Err(c, errors.ErrInvalidPayload)
		return
	}
	items, pag, err := h.roles.ListPermissions(ctx, req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Listed(c, items, pag, nil)
}
