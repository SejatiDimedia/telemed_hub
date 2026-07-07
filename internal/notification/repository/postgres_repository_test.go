package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/model"
)

func setupTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
	)
	require.NoError(t, err)

	connURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	var pool *pgxpool.Pool
	var pingErr error
	for i := 0; i < 20; i++ {
		pool, err = pgxpool.New(ctx, connURL)
		if err == nil {
			pingErr = pool.Ping(ctx)
			if pingErr == nil {
				break
			}
			pool.Close()
		} else {
			pingErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.NoError(t, pingErr, "failed to ping database after retries")

	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	require.NoError(t, err)

	m, err := migrate.New("file://"+migrationsPath, connURL)
	require.NoError(t, err)

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}

	cleanup := func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return pool, cleanup
}

func seedUser(t *testing.T, ctx context.Context, db *pgxpool.Pool) uuid.UUID {
	userID := uuid.New()
	_, err := db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1,$2,$3,$4,$5)`,
		userID, "user@test.com", "hash", "John Doe", "active")
	require.NoError(t, err)
	return userID
}

func TestPostgresRepository_NotificationOperations(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewPostgresRepository(db)
	userID := seedUser(t, ctx, db)

	t.Run("Create & GetByID", func(t *testing.T) {
		id := uuid.New()
		n := &model.Notification{
			ID:        id,
			UserID:    userID,
			Channel:   "email",
			Type:      "appointment_confirmed",
			Status:    "pending",
			Payload:   map[string]any{"key": "value"},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		err := repo.Create(ctx, n)
		assert.NoError(t, err)

		fetched, err := repo.GetByID(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, id, fetched.ID)
		assert.Equal(t, userID, fetched.UserID)
		assert.Equal(t, "email", fetched.Channel)
		assert.Equal(t, "appointment_confirmed", fetched.Type)
		assert.Equal(t, "pending", fetched.Status)
		assert.Equal(t, "value", fetched.Payload["key"])
	})

	t.Run("List User Notifications", func(t *testing.T) {
		// Verify list notifications
		statusStr := "pending"
		list, total, err := repo.List(ctx, userID, &statusStr, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, list, 1)

		// List with nil filter
		list, total, err = repo.List(ctx, userID, nil, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("Update & ListPendingOrFailedEligibleForRetry", func(t *testing.T) {
		id := uuid.New()
		n := &model.Notification{
			ID:        id,
			UserID:    userID,
			Channel:   "email",
			Type:      "order_status",
			Status:    "pending",
			Payload:   map[string]any{"key": "value"},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		err := repo.Create(ctx, n)
		require.NoError(t, err)

		// Update
		now := time.Now().UTC()
		n.Status = "failed"
		n.RetryCount = 1
		n.LastAttemptedAt = &now

		err = repo.Update(ctx, n)
		assert.NoError(t, err)

		// Verify update saved
		fetched, err := repo.GetByID(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, "failed", fetched.Status)
		assert.Equal(t, 1, fetched.RetryCount)
		assert.NotNil(t, fetched.LastAttemptedAt)

		// Get retry eligible notifications
		eligible, err := repo.ListPendingOrFailedEligibleForRetry(ctx, 3)
		assert.NoError(t, err)
		// Should include the updated failed notification (retry_count=1 < 3, and last_attempted_at is older than 30s)
		// Wait, because we set last_attempted_at to "now", the 30-second cutoff check might filter it out!
		// Let's test with a simulated old last_attempted_at.
		oldTime := time.Now().Add(-1 * time.Minute).UTC()
		n.LastAttemptedAt = &oldTime
		err = repo.Update(ctx, n)
		require.NoError(t, err)

		eligible, err = repo.ListPendingOrFailedEligibleForRetry(ctx, 3)
		assert.NoError(t, err)
		assert.NotEmpty(t, eligible)

		found := false
		for _, el := range eligible {
			if el.ID == id {
				found = true
			}
		}
		assert.True(t, found, "Should find eligible failed notification")
	})
}
