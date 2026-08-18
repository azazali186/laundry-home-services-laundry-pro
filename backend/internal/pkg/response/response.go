// Package response provides a uniform JSON envelope for all API responses:
//
//	{
//	  "code": 0, "message": "ok",
//	  "data": {...},
//	  "pagination": {...},
//	  "summary": {...},
//	  "request_id": "..."
//	}
package response

import (
	"context"
	"net/http"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/cloudwego/hertz/pkg/app"
)

// Pagination summarizes offset-based paging results.
type Pagination struct {
	Limit   int   `json:"limit"`
	Total   int64 `json:"total"`
	HasMore bool  `json:"has_more"`
	Offset  int   `json:"offset"`
}

// Envelope is the standard response body.
type Envelope struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	Data       any         `json:"data,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Summary    any         `json:"summary,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
}

const ctxKeyRequestID = "request_id"

// WithRequestID stores the request id in the hertz context.
func WithRequestID(c *app.RequestContext, requestID string) {
	c.Set(ctxKeyRequestID, requestID)
}

// requestID extracts the request id from context.
func requestID(c *app.RequestContext) string {
	if v, ok := c.Get(ctxKeyRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// OK writes a successful response with optional data.
func OK(c *app.RequestContext, data any) {
	Write(c, http.StatusOK, 0, "ok", data, nil, nil)
}

// Created writes a 201 response.
func Created(c *app.RequestContext, data any) {
	Write(c, http.StatusCreated, 0, "created", data, nil, nil)
}

// NoContent writes a 204 response.
func NoContent(c *app.RequestContext) {
	c.SetStatusCode(http.StatusNoContent)
	c.Abort()
}

// Listed writes a paginated list response including summary data.
func Listed(c *app.RequestContext, data any, pag *Pagination, summary any) {
	Write(c, http.StatusOK, 0, "ok", data, pag, summary)
}

// Write is the low-level envelope writer.
func Write(c *app.RequestContext, status, code int, message string, data any, pag *Pagination, summary any) {
	env := Envelope{
		Code:       code,
		Message:    message,
		Data:       data,
		Pagination: pag,
		Summary:    summary,
		RequestID:  requestID(c),
	}
	c.JSON(status, env)
}

// Fail writes a plain failure envelope without an error type.
func Fail(c *app.RequestContext, status, code int, message string) {
	Write(c, status, code, message, nil, nil, nil)
}

// Err maps a domain error to an envelope and writes it.
func Err(c *app.RequestContext, err error) {
	appErr := errors.As(err)
	if appErr == nil {
		appErr = errors.ErrInternal
	}
	Write(c, appErr.Code, appErr.Code, appErr.Message, nil, nil, nil)
}

// AbortErr writes the error envelope and aborts the request chain.
func AbortErr(c *app.RequestContext, err error) {
	Err(c, err)
	c.Abort()
}

// FromContext returns the request id from a standard context.
func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return v
	}
	return ""
}
