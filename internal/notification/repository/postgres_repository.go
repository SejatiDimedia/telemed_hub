package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/model"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

var _ NotificationRepository = (*PostgresRepository)(nil)

func (r *PostgresRepository) Create(ctx context.Context, n *model.Notification) error {
	payloadJSON, err := json.Marshal(n.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := `
		INSERT INTO notifications (
			id, user_id, channel, type, status, payload, retry_count, 
			last_attempted_at, sent_at, failed_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.db.Exec(ctx, query,
		n.ID,
		n.UserID,
		n.Channel,
		n.Type,
		n.Status,
		payloadJSON,
		n.RetryCount,
		n.LastAttemptedAt,
		n.SentAt,
		n.FailedAt,
		n.CreatedAt,
		n.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	query := `
		SELECT 
			id, user_id, channel, type, status, payload, retry_count, 
			last_attempted_at, sent_at, failed_at, created_at, updated_at
		FROM notifications
		WHERE id = $1
	`

	row := r.db.QueryRow(ctx, query, id)

	var n model.Notification
	var payloadBytes []byte
	err := row.Scan(
		&n.ID,
		&n.UserID,
		&n.Channel,
		&n.Type,
		&n.Status,
		&payloadBytes,
		&n.RetryCount,
		&n.LastAttemptedAt,
		&n.SentAt,
		&n.FailedAt,
		&n.CreatedAt,
		&n.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("failed to get notification by id: %w", err)
	}

	if err := json.Unmarshal(payloadBytes, &n.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &n, nil
}

func (r *PostgresRepository) List(ctx context.Context, userID uuid.UUID, status *string, page, limit int) ([]*model.Notification, int, error) {
	offset := (page - 1) * limit

	var countQuery string
	var listQuery string
	var args []any

	if status != nil {
		countQuery = `
			SELECT COUNT(*) FROM notifications 
			WHERE user_id = $1 AND status = $2
		`
		listQuery = `
			SELECT 
				id, user_id, channel, type, status, payload, retry_count, 
				last_attempted_at, sent_at, failed_at, created_at, updated_at
			FROM notifications 
			WHERE user_id = $1 AND status = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`
		args = []any{userID, *status}
	} else {
		countQuery = `
			SELECT COUNT(*) FROM notifications 
			WHERE user_id = $1
		`
		listQuery = `
			SELECT 
				id, user_id, channel, type, status, payload, retry_count, 
				last_attempted_at, sent_at, failed_at, created_at, updated_at
			FROM notifications 
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []any{userID}
	}

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	if total == 0 {
		return []*model.Notification{}, 0, nil
	}

	var listArgs []any
	if status != nil {
		listArgs = append(listArgs, userID, *status, limit, offset)
	} else {
		listArgs = append(listArgs, userID, limit, offset)
	}

	rows, err := r.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var list []*model.Notification
	for rows.Next() {
		var n model.Notification
		var payloadBytes []byte
		err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.Channel,
			&n.Type,
			&n.Status,
			&payloadBytes,
			&n.RetryCount,
			&n.LastAttemptedAt,
			&n.SentAt,
			&n.FailedAt,
			&n.CreatedAt,
			&n.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification row: %w", err)
		}

		if err := json.Unmarshal(payloadBytes, &n.Payload); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		list = append(list, &n)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error during rows iteration: %w", err)
	}

	return list, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, n *model.Notification) error {
	payloadJSON, err := json.Marshal(n.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := `
		UPDATE notifications
		SET 
			status = $1, 
			retry_count = $2, 
			last_attempted_at = $3, 
			sent_at = $4, 
			failed_at = $5, 
			payload = $6,
			updated_at = NOW()
		WHERE id = $7
	`

	res, err := r.db.Exec(ctx, query,
		n.Status,
		n.RetryCount,
		n.LastAttemptedAt,
		n.SentAt,
		n.FailedAt,
		payloadJSON,
		n.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	if res.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *PostgresRepository) ListPendingOrFailedEligibleForRetry(ctx context.Context, maxRetries int) ([]*model.Notification, error) {
	// A notification is eligible for retry if it's:
	// - 'pending' AND hasn't been tried recently (e.g. last_attempted_at is NULL or older than 1 minute)
	// - 'failed' AND retry_count < maxRetries AND last_attempted_at is older than backoff window (we will do a simple check where it's at least 30 seconds old)
	cutoff := time.Now().Add(-30 * time.Second)

	query := `
		SELECT 
			id, user_id, channel, type, status, payload, retry_count, 
			last_attempted_at, sent_at, failed_at, created_at, updated_at
		FROM notifications
		WHERE 
			(status = 'pending' AND (last_attempted_at IS NULL OR last_attempted_at < $1))
			OR (status = 'failed' AND retry_count < $2 AND (last_attempted_at IS NULL OR last_attempted_at < $1))
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, cutoff, maxRetries)
	if err != nil {
		return nil, fmt.Errorf("failed to query retry eligible notifications: %w", err)
	}
	defer rows.Close()

	var list []*model.Notification
	for rows.Next() {
		var n model.Notification
		var payloadBytes []byte
		err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.Channel,
			&n.Type,
			&n.Status,
			&payloadBytes,
			&n.RetryCount,
			&n.LastAttemptedAt,
			&n.SentAt,
			&n.FailedAt,
			&n.CreatedAt,
			&n.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification row: %w", err)
		}

		if err := json.Unmarshal(payloadBytes, &n.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		list = append(list, &n)
	}

	return list, nil
}
