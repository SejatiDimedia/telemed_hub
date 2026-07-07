package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/repository"
)

type MockNotificationRepository struct {
	mock.Mock
}

func (m *MockNotificationRepository) Create(ctx context.Context, n *model.Notification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func (m *MockNotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Notification), args.Error(1)
}

func (m *MockNotificationRepository) List(ctx context.Context, userID uuid.UUID, status *string, page, limit int) ([]*model.Notification, int, error) {
	args := m.Called(ctx, userID, status, page, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*model.Notification), args.Int(1), args.Error(2)
}

func (m *MockNotificationRepository) Update(ctx context.Context, n *model.Notification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func (m *MockNotificationRepository) ListPendingOrFailedEligibleForRetry(ctx context.Context, maxRetries int) ([]*model.Notification, error) {
	args := m.Called(ctx, maxRetries)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Notification), args.Error(1)
}

var _ repository.NotificationRepository = (*MockNotificationRepository)(nil)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	ctx := context.Background()

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	require.NoError(t, err)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: endpoint,
	})

	cleanup := func() {
		rdb.Close()
		_ = redisContainer.Terminate(ctx)
	}

	return rdb, cleanup
}

func TestNotificationService_PublishNotification(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	mockRepo := new(MockNotificationRepository)
	log := slog.New(slog.DiscardHandler)
	svc := NewNotificationService(mockRepo, rdb, log)

	userID := uuid.New()
	payload := map[string]any{"order_id": "123"}

	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

	err := svc.PublishNotification(context.Background(), userID, "email", "order_status", payload)
	assert.NoError(t, err)

	// Verify that it enqueued to Redis Stream
	vals, err := rdb.XRead(context.Background(), &redis.XReadArgs{
		Streams: []string{"telemedhub:notifications:stream", "0-0"},
		Count:   1,
	}).Result()
	assert.NoError(t, err)
	assert.NotEmpty(t, vals)
	mockRepo.AssertExpectations(t)
}

func TestNotificationWorker_RetryLogic(t *testing.T) {
	mockRepo := new(MockNotificationRepository)
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	log := slog.New(slog.DiscardHandler)
	worker := NewNotificationWorker(mockRepo, rdb, log)

	notifID := uuid.New()
	userID := uuid.New()
	n := &model.Notification{
		ID:         notifID,
		UserID:     userID,
		Channel:    "email",
		Type:       "order_status",
		Status:     "pending",
		Payload:    map[string]any{"simulate_failure": true}, // This triggers worker mock dispatch failure
		RetryCount: 0,
	}

	t.Run("Worker dispatch failure increments retry count", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, notifID).Return(n, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(updated *model.Notification) bool {
			return updated.Status == "failed" && updated.RetryCount == 1
		})).Return(nil).Once()

		err := worker.processNotification(context.Background(), notifID)
		assert.Error(t, err)
		assert.Equal(t, "failed", n.Status)
		assert.Equal(t, 1, n.RetryCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Worker marks notification as permanently failed after 3 retries", func(t *testing.T) {
		n.RetryCount = 2 // Already tried 2 times
		mockRepo.On("GetByID", mock.Anything, notifID).Return(n, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(updated *model.Notification) bool {
			return updated.Status == "failed" && updated.RetryCount == 3 && updated.FailedAt != nil
		})).Return(nil).Once()

		err := worker.processNotification(context.Background(), notifID)
		assert.Error(t, err)
		assert.Equal(t, "failed", n.Status)
		assert.Equal(t, 3, n.RetryCount)
		mockRepo.AssertExpectations(t)
	})
}
