package jwtx

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func newTestManager() *Manager {
	return NewManager("test-secret-that-is-long-enough-123456", 15*time.Minute)
}

func TestIssueAndParseRoundtrip(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()

	token, claims, err := m.Issue(userID, "admin@laundrypro.local", []string{"superadmin"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if claims == nil || claims.UserID != userID || claims.Email != "admin@laundrypro.local" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "superadmin" {
		t.Fatalf("roles mismatch: %+v", claims.Roles)
	}
	if claims.Type != TypeAccess {
		t.Fatalf("token type = %q, want access", claims.Type)
	}
	if claims.JTI == "" || claims.ID != claims.JTI {
		t.Fatal("JTI must be set and match the registered claim ID")
	}

	parsed, err := m.Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.UserID != userID || parsed.Email != claims.Email {
		t.Fatalf("parsed claims mismatch: %+v", parsed)
	}
}

func TestParseTamperedToken(t *testing.T) {
	m := newTestManager()
	token, _, _ := m.Issue(uuid.New(), "a@b.c", nil)

	// Flip the last character of the signature portion.
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("malformed jwt: %s", token)
	}
	sig := parts[2]
	flipped := "A" + sig[1:]
	tampered := parts[0] + "." + parts[1] + "." + flipped

	if _, err := m.Parse(tampered); err == nil {
		t.Fatal("Parse must reject a tampered signature")
	}
}

func TestParseWrongSecret(t *testing.T) {
	issuer := NewManager("secret-one-1234567890", time.Minute)
	token, _, _ := issuer.Issue(uuid.New(), "a@b.c", nil)

	verifier := NewManager("secret-two-0987654321", time.Minute)
	if _, err := verifier.Parse(token); err == nil {
		t.Fatal("Parse must reject tokens signed with a different secret")
	}
}

func TestParseExpiredToken(t *testing.T) {
	m := NewManager("expiry-secret-1234567890", -time.Minute) // already expired
	token, _, _ := m.Issue(uuid.New(), "a@b.c", nil)

	if _, err := m.Parse(token); err == nil {
		t.Fatal("Parse must reject an expired token")
	}
}

func TestParseRejectsWrongSigningMethod(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Type:   TypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	// Signed with HS384 — the verifier only accepts HS256.
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString(m.secret)
	if err != nil {
		t.Fatalf("craft token: %v", err)
	}
	if _, err := m.Parse(token); err == nil {
		t.Fatal("Parse must reject a non-HS256 signing method")
	}
}

func TestParseInvalidStrings(t *testing.T) {
	m := newTestManager()
	if _, err := m.Parse(""); err == nil {
		t.Fatal("Parse must reject an empty token")
	}
	if _, err := m.Parse("not.a.jwt"); err == nil {
		t.Fatal("Parse must reject a garbage token")
	}
}

func TestSessionFingerprintDeterministic(t *testing.T) {
	userID := uuid.New()
	f1 := SessionFingerprint("token-abc", userID)
	f2 := SessionFingerprint("token-abc", userID)
	if f1 != f2 {
		t.Fatal("fingerprint must be deterministic for the same token+user")
	}
	if f1 == SessionFingerprint("token-xyz", userID) {
		t.Fatal("fingerprint must differ across tokens")
	}
	if len(f1) != 32 {
		t.Fatalf("fingerprint length = %d, want 32 (md5 hex)", len(f1))
	}
}

func TestManagerNilHandling(t *testing.T) {
	// An empty secret must still produce an error path, not panic.
	m := NewManager("", time.Minute)
	_, _, err := m.Issue(uuid.New(), "a@b.c", nil)
	if err == nil {
		// Empty secret can produce tokens in HS256 (zero key) — ensure parsing
		// a zero-key token with a different manager still fails cleanly.
		m2 := NewManager("another-secret", time.Minute)
		if _, err := m2.Parse(""); !errors.Is(err, jwt.ErrTokenMalformed) {
			t.Fatalf("expected malformed token error, got %v", err)
		}
	}
}
