package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/config"
	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/audit"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/crypto"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/jwtx"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/password"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// sessionTTL is the server-side session lifetime.
const sessionTTL = 24 * time.Hour

// Login brute-force protection: after maxLoginFailures failed attempts within
// lockoutWindow the account is temporarily locked (HTTP 429). Every failure
// counts equally (existing account or not) to avoid user enumeration, and the
// counter is cleared on a successful login. Trade-off: a remote actor can
// temporarily lock a known account (DoS); acceptable here because per-IP and
// per-user rate limiting already bounds request volume before this runs.
const (
	maxLoginFailures = 5
	lockoutWindow    = 15 * time.Minute
)

// AuthService implements authentication flows.
type AuthService struct {
	db     *gorm.DB
	cache  *cache.Cache
	jwt    *jwtx.Manager
	cipher *crypto.Cipher
	cfg    *config.Config
	audit  *audit.Writer
}

// NewAuthService builds the auth service.
func NewAuthService(db *gorm.DB, cache *cache.Cache, jwt *jwtx.Manager, cipher *crypto.Cipher, cfg *config.Config, audit *audit.Writer) *AuthService {
	return &AuthService{db: db, cache: cache, jwt: jwt, cipher: cipher, cfg: cfg, audit: audit}
}

// Login authenticates credentials and creates a session.
func (s *AuthService) Login(ctx context.Context, req LoginRequest, meta audit.Meta) (*LoginResponse, error) {
	// Brute-force guard: reject early once the failure counter is exhausted,
	// without spending a bcrypt comparison (also avoids timing signals).
	failKey := "login:fail:" + strings.ToLower(strings.TrimSpace(req.Email))
	if n, err := s.cache.Incr(ctx, failKey, lockoutWindow); err == nil && n > maxLoginFailures {
		s.logLogin(ctx, meta, uuid.Nil, req.Email, "failed", "account locked")
		return nil, errors.New(429, "too many failed attempts; try again later")
	}

	var user models.User
	err := s.db.WithContext(ctx).Preload("Roles").First(&user, "email = ?", req.Email).Error
	if err != nil {
		s.logLogin(ctx, meta, uuid.Nil, req.Email, "failed", "user not found")
		return nil, errors.ErrInvalidCreds
	}
	if !password.Verify(user.Password, req.Password) {
		s.logLogin(ctx, meta, user.ID, req.Email, "failed", "invalid password")
		return nil, errors.ErrInvalidCreds
	}
	if !user.IsActive {
		s.logLogin(ctx, meta, user.ID, req.Email, "failed", "account disabled")
		return nil, errors.New(403, "account disabled")
	}

	roles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roles = append(roles, r.Code)
	}

	// Successful login clears the failure counter.
	_ = s.cache.Del(ctx, failKey)

	token, _, err := s.jwt.Issue(user.ID, user.Email, roles)
	if err != nil {
		return nil, fmt.Errorf("issue JWT: %w", err)
	}
	// Single sign-on: replace any existing session.
	sessionKey := "session:" + user.ID.String()
	fingerprint := jwtx.SessionFingerprint(token, user.ID)
	if err := s.cache.Set(ctx, sessionKey, fingerprint, sessionTTL); err != nil {
		return nil, fmt.Errorf("store session: %w", err)
	}

	now := time.Now()
	s.db.WithContext(ctx).Model(&user).Update("last_login_at", &now)
	s.logLogin(ctx, meta, user.ID, req.Email, "success", "")

	view, err := s.userView(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{
		AccessToken: token,
		ExpiresIn:   int64(s.cfg.JWTExpiry.Seconds()),
		TokenType:   "Bearer",
		User:        *view,
	}, nil
}

// Logout invalidates the session.
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, meta audit.Meta) error {
	if err := s.cache.Del(ctx, "session:"+userID.String()); err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "user", userID.String(), "logout", nil, map[string]any{"user_id": userID.String()})
	return nil
}

// Me returns the current user profile.
func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (*UserView, error) {
	return s.userView(ctx, userID)
}

// ChangePassword verifies the current password and rotates it.
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest, meta audit.Meta) error {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return errors.ErrNotFound
	}
	if !password.Verify(user.Password, req.CurrentPassword) {
		return errors.ErrInvalidCreds
	}
	hashed, err := password.Hash(req.NewPassword, s.cfg.PasswordCost)
	if err != nil {
		return err
	}
	before := audit.DiffBefore(&user)
	user.Password = hashed
	if err := s.db.WithContext(ctx).Model(&user).Update("password", hashed).Error; err != nil {
		return err
	}
	s.audit.Log(ctx, meta, "user", userID.String(), "update", before, &user)
	return nil
}

// userView loads a user with roles into a public view.
func (s *AuthService) userView(ctx context.Context, userID uuid.UUID) (*UserView, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", userID).Error; err != nil {
		return nil, errors.ErrNotFound
	}
	phone, _ := s.cipher.Decrypt(user.Phone)
	return &UserView{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Phone:       phone,
		IsActive:    user.IsActive,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Roles:       roleViews(user.Roles),
	}, nil
}

func (s *AuthService) logLogin(ctx context.Context, meta audit.Meta, userID uuid.UUID, email, status, reason string) {
	row := models.LoginLog{
		UserID:    userID,
		Email:     email,
		IP:        meta.IP,
		UserAgent: meta.UserAgent,
		Status:    status,
		Reason:    reason,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return
	}
}
