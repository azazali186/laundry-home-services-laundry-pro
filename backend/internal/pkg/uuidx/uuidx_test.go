package uuidx

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewIsUUIDv7(t *testing.T) {
	id := New()
	if id.Version() != 7 {
		t.Fatalf("New() version = %d, want 7", id.Version())
	}
	if id.Variant() != uuid.RFC4122 {
		t.Fatalf("New() variant = %d, want RFC4122", id.Variant())
	}
	// Uniqueness across a small sample.
	seen := map[uuid.UUID]bool{}
	for i := 0; i < 100; i++ {
		id := New()
		if seen[id] {
			t.Fatal("duplicate uuid generated")
		}
		seen[id] = true
	}
}

func TestNewErr(t *testing.T) {
	id, err := NewErr()
	if err != nil {
		t.Fatalf("NewErr: %v", err)
	}
	if id.Version() != 7 {
		t.Fatalf("NewErr version = %d, want 7", id.Version())
	}
}

func TestParse(t *testing.T) {
	id := New()
	canonical := id.String()

	// Canonical roundtrip.
	parsed, err := Parse(canonical)
	if err != nil {
		t.Fatalf("Parse(canonical): %v", err)
	}
	if parsed != id {
		t.Fatal("Parse did not roundtrip the same id")
	}

	// Uppercase form is accepted.
	upper := strings.ToUpper(canonical)
	if _, err := Parse(upper); err != nil {
		t.Fatalf("Parse(upper): %v", err)
	}

	// Surrounding whitespace is trimmed.
	if _, err := Parse("  " + canonical + "\n"); err != nil {
		t.Fatalf("Parse(trimmed): %v", err)
	}

	// Invalid input errors.
	if _, err := Parse("not-a-uuid"); err == nil {
		t.Fatal("Parse(invalid) should error")
	}
	if _, err := Parse(""); err == nil {
		t.Fatal("Parse(empty) should error")
	}
}

func TestStringIsLowercase(t *testing.T) {
	id := New()
	if got := String(id); got != strings.ToLower(got) {
		t.Fatalf("String() = %q, want lowercase", got)
	}
}
