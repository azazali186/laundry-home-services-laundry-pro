// Package gateway abstracts an external payment processor. The production
// implementation is a stub simulating latency; real providers plug in behind
// the same interface. Calls are guarded by circuit breaker + retry.
package gateway

import (
	"context"
	"errors"
	"time"
)

// ChargeResult is returned by a successful charge.
type ChargeResult struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

// ErrDeclined represents a gateway-level decline.
var ErrDeclined = errors.New("payment declined by gateway")

// Client processes payment charges.
type Client interface {
	Charge(ctx context.Context, ref string, amount float64, currency string) (*ChargeResult, error)
	Refund(ctx context.Context, transactionID string, amount float64) error
}

// StubClient simulates a gateway with realistic latency and a deterministic
// decline rule (amounts above 999999 are declined) so flows are testable.
type StubClient struct {
	Latency time.Duration
}

// NewStub builds a stub client with default latency.
func NewStub() *StubClient {
	return &StubClient{Latency: 120 * time.Millisecond}
}

// Charge simulates processing a charge.
func (s *StubClient) Charge(_ context.Context, ref string, amount float64, _ string) (*ChargeResult, error) {
	if s.Latency > 0 {
		time.Sleep(s.Latency)
	}
	if amount > 999999 {
		return nil, ErrDeclined
	}
	return &ChargeResult{
		TransactionID: "TXN-" + ref,
		Status:        "success",
	}, nil
}

// Refund simulates processing a refund.
func (s *StubClient) Refund(_ context.Context, transactionID string, _ float64) error {
	if s.Latency > 0 {
		time.Sleep(s.Latency)
	}
	if transactionID == "" {
		return ErrDeclined
	}
	return nil
}
