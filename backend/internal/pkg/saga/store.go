package saga

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/cache"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Store persists saga state; DB is the source of truth, Redis hot-caches it.
type Store struct {
	db    *gorm.DB
	cache *cache.Cache
}

// NewStore builds the saga state store.
func NewStore(db *gorm.DB, cache *cache.Cache) *Store {
	return &Store{db: db, cache: cache}
}

// Start begins a saga and returns its id.
func (s *Store) Start(ctx context.Context, name, aggregateID string, state any) (string, error) {
	sagaID, err := newSagaID()
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	now := time.Now()
	row := models.SagaState{
		SagaID:      sagaID,
		Name:        name,
		AggregateID: aggregateID,
		Status:      models.SagaRunning,
		Step:        "started",
		State:       raw,
		StartedAt:   now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return "", err
	}
	s.cacheState(ctx, sagaID, &row)
	return sagaID, nil
}

// Advance records progress through a saga step.
func (s *Store) Advance(ctx context.Context, sagaID, step string, state any) error {
	raw, _ := json.Marshal(state)
	return s.update(ctx, sagaID, map[string]any{
		"status": models.SagaRunning,
		"step":   step,
		"state":  raw,
	})
}

// Complete marks a saga successful.
func (s *Store) Complete(ctx context.Context, sagaID string, state any) error {
	raw, _ := json.Marshal(state)
	now := time.Now()
	if err := s.update(ctx, sagaID, map[string]any{
		"status":      models.SagaCompleted,
		"step":        "completed",
		"state":       raw,
		"finished_at": &now,
	}); err != nil {
		return err
	}
	s.cache.Del(ctx, "saga:"+sagaID)
	return nil
}

// Fail marks a saga failed with the given error.
func (s *Store) Fail(ctx context.Context, sagaID string, cause error) error {
	now := time.Now()
	err := s.update(ctx, sagaID, map[string]any{
		"status":      models.SagaFailed,
		"step":        "failed",
		"finished_at": &now,
		"last_error":  cause.Error(),
	})
	s.cache.Del(ctx, "saga:"+sagaID)
	return err
}

// Compensate marks a saga compensated after rolling back.
func (s *Store) Compensate(ctx context.Context, sagaID string, cause error) error {
	now := time.Now()
	err := s.update(ctx, sagaID, map[string]any{
		"status":      models.SagaCompensated,
		"step":        "compensated",
		"finished_at": &now,
		"last_error":  cause.Error(),
	})
	s.cache.Del(ctx, "saga:"+sagaID)
	return err
}

// Get loads saga state, preferring the Redis hot cache.
func (s *Store) Get(ctx context.Context, sagaID string) (*models.SagaState, error) {
	var row models.SagaState
	if err := s.cache.GetJSON(ctx, "saga:"+sagaID, &row); err == nil && row.ID != uuid.Nil {
		return &row, nil
	}
	if err := s.db.WithContext(ctx).Where("saga_id = ?", sagaID).First(&row).Error; err != nil {
		return nil, err
	}
	s.cacheState(ctx, sagaID, &row)
	return &row, nil
}

// FindByAggregate returns the most recent running saga for an aggregate id,
// so flow handlers can advance their own saga from a follow-up event.
func (s *Store) FindByAggregate(ctx context.Context, aggregateID string) (*models.SagaState, error) {
	var row models.SagaState
	if err := s.db.WithContext(ctx).Where("aggregate_id = ? AND status = ?", aggregateID, models.SagaRunning).
		Order("created_at DESC").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Store) update(ctx context.Context, sagaID string, fields map[string]any) error {
	if err := s.db.WithContext(ctx).Model(&models.SagaState{}).Where("saga_id = ?", sagaID).Updates(fields).Error; err != nil {
		return err
	}
	var row models.SagaState
	if err := s.db.WithContext(ctx).Where("saga_id = ?", sagaID).First(&row).Error; err == nil {
		s.cacheState(ctx, sagaID, &row)
	}
	logger.Info("saga %s updated: %v", sagaID, fields["step"])
	return nil
}

func (s *Store) cacheState(ctx context.Context, sagaID string, row *models.SagaState) {
	if err := s.cache.SetJSON(ctx, "saga:"+sagaID, row, 15*time.Minute); err != nil {
		logger.Warn("saga cache write failed: %v", err)
	}
}
