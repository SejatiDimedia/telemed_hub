package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/repository"
)

type NotificationWorker struct {
	repo          repository.NotificationRepository
	rdb           *redis.Client
	log           *slog.Logger
	stream        string
	group         string
	consumerName  string
	maxRetries    int
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	shutdownMutex sync.Mutex
	isStopped     bool
}

func NewNotificationWorker(
	repo repository.NotificationRepository,
	rdb *redis.Client,
	log *slog.Logger,
) *NotificationWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &NotificationWorker{
		repo:         repo,
		rdb:          rdb,
		log:          log,
		stream:       "telemedhub:notifications:stream",
		group:        "notification-group",
		consumerName: "worker-1",
		maxRetries:   3,
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (w *NotificationWorker) Start() {
	w.log.Info("starting notification worker background processing")

	// 1. Create Redis Stream Group if not exists
	err := w.rdb.XGroupCreateMkStream(w.ctx, w.stream, w.group, "$").Err()
	if err != nil {
		// Ignore BUSYGROUP errors
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			w.log.Warn("could not create consumer group, it might already exist", "error", err)
		}
	}

	w.wg.Add(2)
	// Task 1: Redis Stream consumer loop
	go w.consumeLoop()
	// Task 2: DB reconciliation polling loop
	go w.reconcileLoop()
}

func (w *NotificationWorker) Stop() {
	w.shutdownMutex.Lock()
	if w.isStopped {
		w.shutdownMutex.Unlock()
		return
	}
	w.isStopped = true
	w.shutdownMutex.Unlock()

	w.log.Info("shutting down notification worker")
	w.cancel()
	w.wg.Wait()
	w.log.Info("notification worker stopped successfully")
}

func (w *NotificationWorker) consumeLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			// Read from Redis Stream Group (XREADGROUP block 2s)
			entries, err := w.rdb.XReadGroup(w.ctx, &redis.XReadGroupArgs{
				Group:    w.group,
				Consumer: w.consumerName,
				Streams:  []string{w.stream, ">"},
				Count:    1,
				Block:    2 * time.Second,
			}).Result()

			if err != nil {
				if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) {
					continue
				}
				w.log.Error("failed to read from redis stream", "error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, stream := range entries {
				for _, msg := range stream.Messages {
					notifIDStr, ok := msg.Values["notification_id"].(string)
					if !ok {
						w.log.Error("malformed stream message: missing notification_id")
						w.rdb.XAck(w.ctx, w.stream, w.group, msg.ID)
						continue
					}

					notifID, err := uuid.Parse(notifIDStr)
					if err != nil {
						w.log.Error("invalid notification uuid", "error", err, "notification_id", notifIDStr)
						w.rdb.XAck(w.ctx, w.stream, w.group, msg.ID)
						continue
					}

					w.log.Debug("received notification from stream", "notification_id", notifID)
					// Process the notification
					err = w.processNotification(w.ctx, notifID)
					if err == nil {
						// Acknowledge the message
						w.rdb.XAck(w.ctx, w.stream, w.group, msg.ID)
					} else {
						w.log.Warn("notification processing failed, will be retried by reconciliation routine", "notification_id", notifID, "error", err)
						// Acknowledge message anyway to prevent block in stream; retry is handled by DB reconciliation
						w.rdb.XAck(w.ctx, w.stream, w.group, msg.ID)
					}
				}
			}
		}
	}
}

func (w *NotificationWorker) reconcileLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			// List all eligible notifications for retry/reconcile
			list, err := w.repo.ListPendingOrFailedEligibleForRetry(w.ctx, w.maxRetries)
			if err != nil {
				w.log.Error("failed to fetch retry eligible notifications", "error", err)
				continue
			}

			for _, n := range list {
				// Calculate exponential backoff: 2^retry_count * time.Second
				backoff := time.Duration(1<<uint(n.RetryCount)) * time.Second
				if n.LastAttemptedAt != nil && n.LastAttemptedAt.Add(backoff).After(time.Now()) {
					// Too soon to retry
					continue
				}

				w.log.Info("reconciliation routine processing notification", "notification_id", n.ID, "retry_count", n.RetryCount)
				_ = w.processNotification(w.ctx, n.ID)
			}
		}
	}
}

func (w *NotificationWorker) processNotification(ctx context.Context, id uuid.UUID) error {
	n, err := w.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to fetch notification: %w", err)
	}

	if n.Status == "sent" || n.Status == "read" {
		return nil // Already processed
	}

	now := time.Now()
	n.LastAttemptedAt = &now
	n.RetryCount++

	// Dispatch notification (simulated email/sms sending)
	err = w.dispatch(n)
	if err == nil {
		n.Status = "sent"
		n.SentAt = &now
		w.log.Info("notification sent successfully", "notification_id", id, "channel", n.Channel, "type", n.Type)
	} else {
		w.log.Error("failed to dispatch notification attempt", "notification_id", id, "retry_count", n.RetryCount, "error", err)
		if n.RetryCount >= w.maxRetries {
			n.Status = "failed"
			n.FailedAt = &now
			w.log.Error("notification reached max retries, marked as failed permanent", "notification_id", id)
		} else {
			n.Status = "failed" // DB status stays failed for retry pickup
		}
	}

	if dbErr := w.repo.Update(ctx, n); dbErr != nil {
		w.log.Error("failed to update notification status in DB", "error", dbErr, "notification_id", id)
		return dbErr
	}

	return err
}

func (w *NotificationWorker) dispatch(n *model.Notification) error {
	// Simulate external service dispatching (e.g. SMTP server)
	// To test retry logic, we can check for a special simulated fail trigger in the payload
	if n.Payload != nil {
		if val, exists := n.Payload["simulate_failure"]; exists {
			if simulate, ok := val.(bool); ok && simulate {
				return errors.New("simulated external provider network error")
			}
		}
	}

	// Simple log dispatching
	w.log.Info("DISPATCHED NOTIFICATION CHANNEL",
		"channel", n.Channel,
		"type", n.Type,
		"user_id", n.UserID,
		"payload", n.Payload,
	)
	return nil
}
