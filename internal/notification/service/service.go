package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/dto"
)

var (
	ErrUnauthorized = errors.New("unauthorized access to notification")
	ErrNotFound     = errors.New("notification not found")
)

type NotificationService interface {
	PublishNotification(ctx context.Context, userID uuid.UUID, channel string, typeStr string, payload map[string]any) error
	ListNotifications(ctx context.Context, userID uuid.UUID, status *string, page, limit int) ([]*dto.NotificationResponse, int, error)
	MarkAsRead(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
}
