package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/model"
)

var ErrNotificationNotFound = errors.New("notification not found")

type NotificationRepository interface {
	Create(ctx context.Context, n *model.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	List(ctx context.Context, userID uuid.UUID, status *string, page, limit int) ([]*model.Notification, int, error)
	Update(ctx context.Context, n *model.Notification) error
	ListPendingOrFailedEligibleForRetry(ctx context.Context, maxRetries int) ([]*model.Notification, error)
}
