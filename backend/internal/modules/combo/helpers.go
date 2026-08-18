package combo

import (
	"context"
	"strings"

	"github.com/aeroxe/laundry-pro/backend/internal/models"
	"github.com/aeroxe/laundry-pro/backend/internal/pkg/errors"
	"github.com/google/uuid"
)

// validateItems ensures quantities are positive and every referenced service
// type exists.
func (s *Service) validateItems(ctx context.Context, items []ItemRequest) error {
	ids := make([]uuid.UUID, 0, len(items))
	seen := map[uuid.UUID]bool{}
	for _, it := range items {
		if it.ServiceTypeID == uuid.Nil {
			return errors.New(400, "item service type is required")
		}
		if it.Quantity < 1 {
			return errors.New(400, "item quantity must be at least 1")
		}
		if !seen[it.ServiceTypeID] {
			seen[it.ServiceTypeID] = true
			ids = append(ids, it.ServiceTypeID)
		}
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.ServiceType{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return errors.New(400, "one or more service types do not exist")
	}
	return nil
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func isUnique(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "duplicate key value")
}
