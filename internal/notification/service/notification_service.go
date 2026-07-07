package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/repository"
)

type NotificationServiceImpl struct {
	repo   repository.NotificationRepository
	rdb    *redis.Client
	log    *slog.Logger
	stream string
}

func NewNotificationService(
	repo repository.NotificationRepository,
	rdb *redis.Client,
	log *slog.Logger,
) *NotificationServiceImpl {
	return &NotificationServiceImpl{
		repo:   repo,
		rdb:    rdb,
		log:    log,
		stream: "telemedhub:notifications:stream",
	}
}

var _ NotificationService = (*NotificationServiceImpl)(nil)

func (s *NotificationServiceImpl) PublishNotification(
	ctx context.Context,
	userID uuid.UUID,
	channel string,
	typeStr string,
	payload map[string]any,
) error {
	n := &model.Notification{
		ID:         uuid.New(),
		UserID:     userID,
		Channel:    channel,
		Type:       typeStr,
		Status:     "pending",
		Payload:    payload,
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 1. Persist to DB
	if err := s.repo.Create(ctx, n); err != nil {
		return fmt.Errorf("failed to persist notification: %w", err)
	}

	// 2. Publish to Redis Stream
	err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: s.stream,
		Values: map[string]any{
			"notification_id": n.ID.String(),
		},
	}).Err()
	if err != nil {
		// Log the error but don't fail, since background reconciliation worker will pick it up
		s.log.Error("failed to publish notification to Redis Stream", "error", err, "notification_id", n.ID)
	}

	return nil
}

func (s *NotificationServiceImpl) ListNotifications(
	ctx context.Context,
	userID uuid.UUID,
	status *string,
	page, limit int,
) ([]*dto.NotificationResponse, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	list, total, err := s.repo.List(ctx, userID, status, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}

	resp := make([]*dto.NotificationResponse, len(list))
	for i, n := range list {
		resp[i] = &dto.NotificationResponse{
			ID:              n.ID.String(),
			UserID:          n.UserID.String(),
			Channel:         n.Channel,
			Type:            n.Type,
			Status:          n.Status,
			Payload:         n.Payload,
			RetryCount:      n.RetryCount,
			LastAttemptedAt: n.LastAttemptedAt,
			SentAt:          n.SentAt,
			FailedAt:        n.FailedAt,
			CreatedAt:       n.CreatedAt,
		}
	}

	return resp, total, nil
}

func (s *NotificationServiceImpl) MarkAsRead(
	ctx context.Context,
	userID uuid.UUID,
	id uuid.UUID,
) error {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to fetch notification: %w", err)
	}

	if n.UserID != userID {
		return ErrUnauthorized
	}

	// Change status to read
	n.Status = "read"
	if err := s.repo.Update(ctx, n); err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}
