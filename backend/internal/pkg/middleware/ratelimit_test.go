package middleware

import "testing"

func TestLimitForRouteSelection(t *testing.T) {
	routes := []RouteLimit{
		{Prefix: "/api/v1/auth/", Limit: 30},
		{Prefix: "/api/v1/bookings", Limit: 60},
	}
	global := int64(120)

	cases := []struct {
		path string
		want int64
	}{
		{"/api/v1/auth/change-password", 30},
		{"/api/v1/bookings", 60},
		{"/api/v1/bookings/123", 60},
		{"/api/v1/orders/list", 120}, // list suffix -> global
		{"/api/v1/service-types/list", 120},
		{"/api/v1/wishlist", 120},  // false-positive guard: no /list suffix
		{"/api/v1/customers", 120}, // no match -> global
	}
	for _, c := range cases {
		if got := limitFor(c.path, routes, global); got != c.want {
			t.Errorf("limitFor(%q) = %d, want %d", c.path, got, c.want)
		}
	}
}

func TestLimitForLongestPrefixWins(t *testing.T) {
	routes := []RouteLimit{
		{Prefix: "/api/v1/", Limit: 50},
		{Prefix: "/api/v1/bookings", Limit: 60},
	}
	if got := limitFor("/api/v1/bookings", routes, 100); got != 60 {
		t.Fatalf("longest prefix must win, got %d", got)
	}
}
