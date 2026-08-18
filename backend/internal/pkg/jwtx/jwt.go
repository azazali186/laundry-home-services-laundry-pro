// Package jwtx issues and validates HS256 JWTs with a Redis-backed session
// for single-sign-on semantics (one active token per user, sliding renewal).
package jwtx

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType distinguishes access tokens.
type TokenType string

const (
	TypeAccess TokenType = "access"
)

// Claims are the JWT claims carried by tokens.
type Claims struct {
	UserID uuid.UUID `json:"uid"`
	Email  string    `json:"email"`
	Roles  []string  `json:"roles"`
	JTI    string    `json:"jti"`
	Type   TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// Manager issues, parses and validates tokens.
type Manager struct {
	secret []byte
	expiry time.Duration
}

// NewManager builds a token manager.
func NewManager(secret string, expiry time.Duration) *Manager {
	return &Manager{secret: []byte(secret), expiry: expiry}
}

// Issue creates a signed access token for the given identity.
func (m *Manager) Issue(userID uuid.UUID, email string, roles []string) (string, *Claims, error) {
	now := time.Now()
	jti := fmt.Sprintf("%s-%d", userID.String(), now.UnixNano())
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		JTI:    jti,
		Type:   TypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "laundry-pro",
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiry)),
			ID:        jti,
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", nil, err
	}
	return token, claims, nil
}

// Parse validates a token string and returns its claims.
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		// Strictly HS256 only — rejects HS384/HS512 tokens (alg confusion) even
		// though they share the HMAC family.
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.Type != TypeAccess {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// SessionFingerprint derives the session identifier stored in Redis:
// md5(token || user_id), matching the reference single-sign-on flow.
func SessionFingerprint(tokenStr string, userID uuid.UUID) string {
	sum := md5.Sum([]byte(tokenStr + userID.String()))
	return hex.EncodeToString(sum[:])
}
