//go:build integration

// Package tests contains end-to-end integration tests that exercise a
// running LaundryPro server over HTTP. They are excluded from the default
// unit-test run (see the `integration` build tag) and require a live stack:
//
//	BASE_URL=http://localhost:9090/api/v1 \
//	ADMIN_EMAIL=admin@laundrypro.local ADMIN_PASSWORD=Admin@12345 \
//	go test ./tests/... -tags=integration -count=1 -v
//
// The Docker compose stack (make up-all) provides Postgres, Redis and NATS.
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// httpClient fails fast instead of hanging the test when a server stalls.
var httpClient = &http.Client{Timeout: 5 * time.Second}

func baseURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("BASE_URL")
	if u == "" {
		u = "http://localhost:9090/api/v1"
	}
	return u
}

func post(t *testing.T, url, token string, body any, want int) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		t.Fatalf("POST %s = %d, want %d", url, resp.StatusCode, want)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	return out
}

func TestReadiness(t *testing.T) {
	url := baseURL(t) + "/health/ready"
	body := post(t, url, "", map[string]any{}, http.StatusOK)
	checks, ok := body["data"].(map[string]any)["checks"].(map[string]any)
	if !ok {
		t.Fatalf("readiness payload shape: %v", body)
	}
	for _, dep := range []string{"cache", "database"} {
		if checks[dep] != "ok" {
			t.Fatalf("dependency %s not ok: %v", dep, checks)
		}
	}
}

func TestLoginAndBookingFlow(t *testing.T) {
	login := post(t, baseURL(t)+"/auth/login", "", map[string]any{
		"email":    getenv(t, "ADMIN_EMAIL", "admin@laundrypro.local"),
		"password": getenv(t, "ADMIN_PASSWORD", "Admin@12345"),
	}, http.StatusOK)
	data, ok := login["data"].(map[string]any)
	if !ok {
		t.Fatalf("login payload: %v", login)
	}
	token, _ := data["access_token"].(string)
	if token == "" {
		t.Fatal("no access_token in login response")
	}

	// Read paths: the bootstrap admin doubles as the customer; service types
	// come from the seeded list. IDs drive the booking create validation.
	me := post(t, baseURL(t)+"/auth/me", token, map[string]any{}, http.StatusOK)
	custID, _ := me["data"].(map[string]any)["id"].(string)
	st := post(t, baseURL(t)+"/service-types/list", token, map[string]any{"limit": 1}, http.StatusOK)
	stID := firstID(t, st)

	// Write path: create a booking (enqueues an outbox event -> saga step).
	booking := post(t, baseURL(t)+"/bookings", token, map[string]any{
		"customer_id":      custID,
		"service_type_id":  stID,
		"pickup_address":   "Integration Test Ave",
		"estimated_amount": 100,
		"currency":         "INR",
	}, http.StatusCreated)
	if _, ok := booking["data"].(map[string]any); !ok {
		t.Fatalf("booking payload: %v", booking)
	}
}

func firstID(t *testing.T, body map[string]any) string {
	t.Helper()
	// List responses carry data as an array: data: [{...}], pagination: {...}
	items, ok := body["data"].([]any)
	if !ok || len(items) == 0 {
		return "" // empty list: leave ids empty and let validation catch it
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := first["id"].(string)
	return id
}

func patch(t *testing.T, url, token string, body any, want int) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		t.Fatalf("PATCH %s = %d, want %d", url, resp.StatusCode, want)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	return out
}

func getenv(t *testing.T, key, fallback string) string {
	t.Helper()
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestLoginRejectsBadPassword(t *testing.T) {
	// Use an isolated email: failed attempts accumulate on the account's
	// lockout counter (5 fails / 15min), so hitting the shared admin here
	// would eventually lock it and poison every other test in the suite.
	email := fmt.Sprintf("badpass-%d@x.local", time.Now().UnixNano())
	post(t, baseURL(t)+"/auth/login", "", map[string]any{
		"email":    email,
		"password": "definitely-wrong",
	}, http.StatusUnauthorized)
}

// TestSettingsFlow covers the shop-configuration endpoints end-to-end and
// proves the settings defaults actually flow into money-flow modules: after
// switching the default currency to USD, a booking created without a currency
// must be stored as USD.
func TestSettingsFlow(t *testing.T) {
	base := baseURL(t)
	token := adminToken(t)

	// View lazily returns the singleton row.
	view := post(t, base+"/settings/view", token, map[string]any{}, http.StatusOK)
	data, ok := view["data"].(map[string]any)
	if !ok {
		t.Fatalf("settings view payload: %v", view)
	}
	if data["business_name"] != "My Laundry" || data["currency"] != "INR" {
		t.Fatalf("unexpected default settings: %v", data)
	}

	// Patch normalizes currency and tax rate.
	patched := patch(t, base+"/settings", token, map[string]any{
		"currency": "usd", "tax_rate": 5,
	}, http.StatusOK)
	pdata, _ := patched["data"].(map[string]any)
	if pdata["currency"] != "USD" || pdata["tax_rate"] != float64(5) {
		t.Fatalf("patch not normalized/persisted: %v", pdata)
	}
	t.Cleanup(func() {
		// Restore neutral defaults so other tests are unaffected.
		patch(t, base+"/settings", token, map[string]any{
			"business_name": "My Laundry", "currency": "INR", "tax_rate": 0,
			"invoice_prefix": "INV", "timezone": "Asia/Kolkata",
		}, http.StatusOK)
	})

	// Validation: tax rate over 100 is rejected with 400.
	patch(t, base+"/settings", token, map[string]any{"tax_rate": 150}, http.StatusBadRequest)

	// Integration proof: a booking created without a currency inherits USD.
	me := post(t, base+"/auth/me", token, map[string]any{}, http.StatusOK)
	custID, _ := me["data"].(map[string]any)["id"].(string)
	st := post(t, base+"/service-types/list", token, map[string]any{"limit": 1}, http.StatusOK)
	stID := firstID(t, st)
	booking := post(t, base+"/bookings", token, map[string]any{
		"customer_id":      custID,
		"service_type_id":  stID,
		"pickup_address":   "Settings Flow Ave",
		"estimated_amount": 100,
	}, http.StatusCreated)
	bdata, _ := booking["data"].(map[string]any)
	if bdata["currency"] != "USD" {
		t.Fatalf("booking currency = %v, want shop default USD (settings not wired into bookings)", bdata["currency"])
	}
}

// waitFor polls fn until it returns true or the timeout expires.
func waitFor(t *testing.T, timeout time.Duration, desc string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", desc)
}

// listData fetches a page of a list endpoint. filters narrows the query via
// the backend's filterable columns (e.g. booking_id) so lookups stay precise
// even as test data accumulates across runs.
func listData(t *testing.T, url, token string, filters map[string]any) []any {
	t.Helper()
	body := map[string]any{"limit": 50, "offset": 0}
	if len(filters) > 0 {
		body["filters"] = filters
	}
	resp := post(t, url, token, body, http.StatusOK)
	items, _ := resp["data"].([]any)
	return items
}

// TestFullBookingSaga drives the entire BookingFlow saga over the HTTP API:
// booking -> pickup -> order -> processing -> quality_check -> ready ->
// delivery -> delivered -> invoice. It proves the outbox relay + saga engine
// advance end-to-end against the live stack and that the shop-configured
// invoice prefix lands on the saga-issued invoice.
func TestFullBookingSaga(t *testing.T) {
	base := baseURL(t)
	token := adminToken(t)

	// Pin a distinctive prefix so the shop setting is provably honored.
	patch(t, base+"/settings", token, map[string]any{"invoice_prefix": "E2E"}, http.StatusOK)
	t.Cleanup(func() {
		patch(t, base+"/settings", token, map[string]any{
			"business_name": "My Laundry", "currency": "INR", "tax_rate": 0,
			"invoice_prefix": "INV", "timezone": "Asia/Kolkata",
		}, http.StatusOK)
	})

	me := post(t, base+"/auth/me", token, map[string]any{}, http.StatusOK)
	custID, _ := me["data"].(map[string]any)["id"].(string)
	st := post(t, base+"/service-types/list", token, map[string]any{"limit": 1}, http.StatusOK)
	stID := firstID(t, st)

	// Step 1: booking created -> saga starts, relay schedules a pickup.
	booking := post(t, base+"/bookings", token, map[string]any{
		"customer_id": custID, "service_type_id": stID,
		"pickup_address": "Saga E2E Ave", "estimated_amount": 120,
	}, http.StatusCreated)
	bookingID, _ := booking["data"].(map[string]any)["id"].(string)
	if bookingID == "" {
		t.Fatal("no booking id in response")
	}

	// Step 2: relay picks up booking_created -> pickup exists for the booking.
	var pickupID string
	waitFor(t, 20*time.Second, "pickup created by relay", func() bool {
		for _, it := range listData(t, base+"/pickups/list", token, map[string]any{"booking_id": bookingID}) {
			m, _ := it.(map[string]any)
			if id, _ := m["id"].(string); id != "" {
				pickupID = id
				return true
			}
		}
		return false
	})

	// Step 3: complete pickup -> order created for the booking.
	post(t, base+"/pickups/complete", token, map[string]any{"id": pickupID}, http.StatusOK)
	var orderID string
	waitFor(t, 20*time.Second, "order created for booking", func() bool {
		for _, it := range listData(t, base+"/orders/list", token, map[string]any{"booking_id": bookingID}) {
			m, _ := it.(map[string]any)
			if id, _ := m["id"].(string); id != "" {
				orderID = id
				return true
			}
		}
		return false
	})

	// Step 4: walk the status pipeline to ready; the ready transition is what
	// schedules the delivery (saga step 3).
	for _, status := range []string{"processing", "quality_check", "ready"} {
		post(t, base+"/orders/status", token, map[string]any{"id": orderID, "status": status}, http.StatusOK)
	}
	var deliveryID string
	waitFor(t, 20*time.Second, "delivery scheduled for order", func() bool {
		for _, it := range listData(t, base+"/deliveries/list", token, map[string]any{"order_id": orderID}) {
			m, _ := it.(map[string]any)
			if id, _ := m["id"].(string); id != "" {
				deliveryID = id
				return true
			}
		}
		return false
	})

	// Step 5: complete delivery -> invoice issued with the shop prefix.
	post(t, base+"/deliveries/complete", token, map[string]any{"id": deliveryID}, http.StatusOK)
	waitFor(t, 20*time.Second, "invoice issued for order", func() bool {
		for _, it := range listData(t, base+"/invoices/list", token, map[string]any{"order_id": orderID}) {
			m, _ := it.(map[string]any)
			no, _ := m["invoice_no"].(string)
			if !strings.HasPrefix(no, "E2E-") {
				t.Fatalf("invoice_no = %q, want shop prefix E2E- (settings not honored by saga)", no)
			}
			return true
		}
		return false
	})

	// The order must reach the delivered terminal state.
	waitFor(t, 10*time.Second, "order delivered", func() bool {
		for _, it := range listData(t, base+"/orders/list", token, map[string]any{"booking_id": bookingID}) {
			m, _ := it.(map[string]any)
			if m["status"] == "delivered" {
				return true
			}
		}
		return false
	})
}

func adminToken(t *testing.T) string {
	t.Helper()
	login := post(t, baseURL(t)+"/auth/login", "", map[string]any{
		"email":    getenv(t, "ADMIN_EMAIL", "admin@laundrypro.local"),
		"password": getenv(t, "ADMIN_PASSWORD", "Admin@12345"),
	}, http.StatusOK)
	data, ok := login["data"].(map[string]any)
	if !ok {
		t.Fatalf("login payload: %v", login)
	}
	token, _ := data["access_token"].(string)
	if token == "" {
		t.Fatal("no access_token in login response")
	}
	return token
}

func TestUnauthenticatedAccessDenied(t *testing.T) {
	resp, err := httpClient.Post(baseURL(t)+"/bookings/list", "application/json",
		bytes.NewReader([]byte(`{"limit":1}`)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /bookings/list = %d, want 401", resp.StatusCode)
	}
}
