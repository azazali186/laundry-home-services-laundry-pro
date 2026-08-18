package auth

import (
	"context"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/middleware"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// Handler exposes auth endpoints.
type Handler struct {
	auth  *AuthService
	users *UserService
	roles *RoleService
	logs  *LogService
}

// NewHandler builds the auth handler.
func NewHandler(auth *AuthService, users *UserService, roles *RoleService, logs *LogService) *Handler {
	return &Handler{auth: auth, users: users, roles: roles, logs: logs}
}

// Login handles POST /api/v1/auth/login
//
//	@Summary	Authenticate a user
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body	auth.LoginRequest	true	"Credentials"
//	@Success	200		{object}	response.Envelope
//	@Failure	401		{object}	response.Envelope
//	@Router		/auth/login [post]
func (h *Handler) Login(ctx context.Context, c *app.RequestContext) {
	var req LoginRequest
	if err := c.Bind(&req); err != nil || req.Email == "" || req.Password == "" {
		response.Err(c, errors.New(400, "email and password are required"))
		return
	}
	result, err := h.auth.Login(ctx, req, audit.MetaFrom(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, result)
}

// Logout handles POST /api/v1/auth/logout
//
//	@Summary	Invalidate the current session
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.Envelope
//	@Router		/auth/logout [post]
func (h *Handler) Logout(ctx context.Context, c *app.RequestContext) {
	userID, err := middleware.UserID(c)
	if err != nil {
		response.Err(c, err)
		return
	}
	if err := h.auth.Logout(ctx, userID, audit.MetaFrom(c)); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, map[string]bool{"logged_out": true})
}

// Me handles POST /api/v1/auth/me
//
//	@Summary	Current user profile
//	@Tags		auth
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.Envelope
//	@Router		/auth/me [post]
func (h *Handler) Me(ctx context.Context, c *app.RequestContext) {
	userID, err := middleware.UserID(c)
	if err != nil {
		response.Err(c, err)
		return
	}
	view, err := h.auth.Me(ctx, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, view)
}

// ChangePassword handles POST /api/v1/auth/change-password
//
//	@Summary	Rotate the current user's password
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body	auth.ChangePasswordRequest	true	"Passwords"
//	@Success	200		{object}	response.Envelope
//	@Router		/auth/change-password [post]
func (h *Handler) ChangePassword(ctx context.Context, c *app.RequestContext) {
	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil || req.CurrentPassword == "" || len(req.NewPassword) < 8 {
		response.Err(c, errors.New(400, "current password and a new password (min 8 chars) are required"))
		return
	}
	userID, err := middleware.UserID(c)
	if err != nil {
		response.Err(c, err)
		return
	}
	if err := h.auth.ChangePassword(ctx, userID, req, audit.MetaFrom(c)); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, map[string]bool{"changed": true})
}
