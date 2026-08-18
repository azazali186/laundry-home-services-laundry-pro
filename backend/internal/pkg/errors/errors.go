// Package errors defines domain-level error types used across the backend.
// HTTP status codes are embedded so handlers can map errors consistently.
package errors

import (
	"errors"
	"net/http"
)

// AppError is a domain error carrying an HTTP status code and a public message.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *AppError) Error() string { return e.Message }

// New creates a new AppError.
func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// WithDetail attaches internal detail to the error (not exposed to clients).
func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}

// Common domain errors.
var (
	ErrBadRequest     = New(http.StatusBadRequest, "invalid request")
	ErrUnauthorized   = New(http.StatusUnauthorized, "unauthorized")
	ErrForbidden      = New(http.StatusForbidden, "forbidden")
	ErrNotFound       = New(http.StatusNotFound, "resource not found")
	ErrConflict       = New(http.StatusConflict, "resource conflict")
	ErrInternal       = New(http.StatusInternalServerError, "internal server error")
	ErrTooMany        = New(http.StatusTooManyRequests, "too many requests")
	ErrInvalidToken   = New(http.StatusUnauthorized, "invalid token")
	ErrTokenExpired   = New(http.StatusUnauthorized, "token expired")
	ErrInvalidCreds   = New(http.StatusUnauthorized, "invalid credentials")
	ErrEmailInUse     = New(http.StatusConflict, "email already in use")
	ErrInvalidPayload = New(http.StatusBadRequest, "invalid payload")
	ErrLimitExceeded  = New(413, "request body too large")
)

// As unwraps an AppError from err, returning nil when not found.
func As(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// IsNotFound reports whether the error is a not-found style error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
