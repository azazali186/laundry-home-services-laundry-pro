package modkit

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/pagination"
	"github.com/google/uuid"
)

var noPattern = regexp.MustCompile(`^[A-Z0-9]{10}$`)

func TestNextNoFormat(t *testing.T) {
	no := NextNo("ORD")
	if !strings.HasPrefix(no, "ORD-") {
		t.Fatalf("NextNo = %q, want ORD- prefix", no)
	}
	suffix := strings.TrimPrefix(no, "ORD-")
	if !noPattern.MatchString(suffix) {
		t.Fatalf("NextNo suffix %q does not match 10 uppercase alphanumerics", suffix)
	}
}

func TestNextNoUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		no := NextNo("INV")
		if seen[no] {
			t.Fatalf("duplicate number %q", no)
		}
		seen[no] = true
	}
}

func TestIsUnique(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("UNIQUE constraint failed: combos.code"), true},
		{errors.New(`duplicate key value violates unique constraint "combos_code_key"`), true},
		{errors.New(`pq: duplicate key value violates unique constraint "users_email_key"`), true},
		{errors.New("SQLSTATE 23505"), true},
		{errors.New("some other db error"), false},
		{nil, false},
	}
	for _, tc := range cases {
		if got := IsUnique(tc.err); got != tc.want {
			t.Errorf("IsUnique(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestOrDefault(t *testing.T) {
	if got := OrDefault("", "fallback"); got != "fallback" {
		t.Fatalf("OrDefault(empty) = %q", got)
	}
	if got := OrDefault("   ", "fallback"); got != "fallback" {
		t.Fatalf("OrDefault(blank) = %q", got)
	}
	if got := OrDefault("real", "fallback"); got != "real" {
		t.Fatalf("OrDefault(present) = %q", got)
	}
}

func TestPageRes(t *testing.T) {
	if got := PageRes(nil); got != nil {
		t.Fatal("PageRes(nil) must return nil")
	}
	res := PageRes(&pagination.PageResult{
		Limit:   10,
		Total:   42,
		HasMore: true,
		Offset:  20,
	})
	if res == nil {
		t.Fatal("PageRes should not return nil for a real result")
	}
	if res.Limit != 10 || res.Total != 42 || !res.HasMore || res.Offset != 20 {
		t.Fatalf("PageRes mismatch: %+v", res)
	}
}

func TestNil(t *testing.T) {
	if Nil() != uuid.Nil {
		t.Fatal("Nil() must return the zero UUID")
	}
}
