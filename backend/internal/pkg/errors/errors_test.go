package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewAndWithDetail(t *testing.T) {
	e := New(http.StatusBadRequest, "bad")
	if e.Code != http.StatusBadRequest || e.Message != "bad" {
		t.Fatalf("unexpected AppError: %+v", e)
	}
	if e.Error() != "bad" {
		t.Fatalf("Error() = %q, want %q", e.Error(), "bad")
	}
	detailed := e.WithDetail("field x invalid")
	if detailed.Detail != "field x invalid" {
		t.Fatalf("Detail not set: %+v", detailed)
	}
	// WithDetail mutates in place.
	if e.Detail != "field x invalid" {
		t.Fatal("WithDetail must mutate the receiver")
	}
}

func TestCommonErrorsCarryStatusCodes(t *testing.T) {
	cases := []struct {
		err     *AppError
		code    int
		message string
	}{
		{ErrBadRequest, http.StatusBadRequest, "invalid request"},
		{ErrUnauthorized, http.StatusUnauthorized, "unauthorized"},
		{ErrForbidden, http.StatusForbidden, "forbidden"},
		{ErrNotFound, http.StatusNotFound, "resource not found"},
		{ErrConflict, http.StatusConflict, "resource conflict"},
		{ErrInternal, http.StatusInternalServerError, "internal server error"},
		{ErrTooMany, http.StatusTooManyRequests, "too many requests"},
		{ErrInvalidToken, http.StatusUnauthorized, "invalid token"},
		{ErrTokenExpired, http.StatusUnauthorized, "token expired"},
		{ErrInvalidCreds, http.StatusUnauthorized, "invalid credentials"},
		{ErrEmailInUse, http.StatusConflict, "email already in use"},
		{ErrInvalidPayload, http.StatusBadRequest, "invalid payload"},
		{ErrLimitExceeded, 413, "request body too large"},
	}
	for _, tc := range cases {
		if tc.err.Code != tc.code || tc.err.Message != tc.message {
			t.Errorf("%q: got (%d, %q), want (%d, %q)", tc.message, tc.err.Code, tc.err.Message, tc.code, tc.message)
		}
	}
}

func TestAs(t *testing.T) {
	if got := As(New(http.StatusNotFound, "nope")); got == nil {
		t.Fatal("As should unwrap a direct AppError")
	}
	wrapped := errors.Join(errors.New("outer"), ErrConflict)
	if got := As(wrapped); got == nil || got != ErrConflict {
		t.Fatalf("As should find wrapped AppError, got %v", got)
	}
	if got := As(errors.New("plain")); got != nil {
		t.Fatalf("As should return nil for non-AppError, got %v", got)
	}
	if got := As(nil); got != nil {
		t.Fatal("As should return nil for nil error")
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(ErrNotFound) {
		t.Fatal("ErrNotFound should satisfy IsNotFound")
	}
	if !IsNotFound(errors.Join(ErrNotFound, errors.New("wrapped"))) {
		t.Fatal("wrapped ErrNotFound should satisfy IsNotFound")
	}
	if IsNotFound(ErrBadRequest) {
		t.Fatal("ErrBadRequest must not satisfy IsNotFound")
	}
}
