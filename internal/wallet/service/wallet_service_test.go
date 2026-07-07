package service

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
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/repository"
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

func seedPatientUser(t *testing.T, ctx context.Context, db *pgxpool.Pool) (userID, patientID uuid.UUID) {
	userID = uuid.New()
	patientID = uuid.New()

	var roleID uuid.UUID
	err := db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'patient'`).Scan(&roleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1,$2,$3,$4,$5)`,
		userID, "walletsvc@test.com", "hash", "Pat Wallet Svc", "active")
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2)`, userID, roleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO patients (id, user_id) VALUES ($1,$2)`, patientID, userID)
	require.NoError(t, err)

	return userID, patientID
}

func TestWalletService_Operations(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repo := repository.NewPostgresRepository(db)
	svc := NewWalletService(repo, db)

	userID, _ := seedPatientUser(t, ctx, db)

	t.Run("GetBalance for new wallet defaults to 0", func(t *testing.T) {
		bal, err := svc.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), bal)
	})

	t.Run("TopUp succeeds and records ledger entry", func(t *testing.T) {
		key := "topup-key-1"
		resp, err := svc.TopUp(ctx, userID, 100000.00, &key)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 100000.00, resp.Amount)
		assert.Equal(t, 100000.00, resp.BalanceAfter)
		assert.Equal(t, "top_up", resp.Type)

		// Second top up with same idempotency key returns cached response
		respDuplicate, err := svc.TopUp(ctx, userID, 100000.00, &key)
		assert.NoError(t, err)
		assert.Equal(t, resp.ID, respDuplicate.ID)

		// Check updated balance
		bal, err := svc.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, int64(100000), bal)
	})

	t.Run("TopUp exceeds max limit fails", func(t *testing.T) {
		// Set env variable to trigger custom limit check
		t.Setenv("WALLET_MAX_TOPUP_AMOUNT", "50000")
		svcWithLimit := NewWalletService(repo, db)

		_, err := svcWithLimit.TopUp(ctx, userID, 60000.00, nil)
		assert.ErrorIs(t, err, ErrMaxTopUpExceeded)
	})

	t.Run("Deduct succeeds and Refund restores balance", func(t *testing.T) {
		// Deduct 40000
		err := svc.Deduct(ctx, userID, 40000, "Order 1 payment")
		assert.NoError(t, err)

		bal, err := svc.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, int64(60000), bal)

		// Refund 20000
		err = svc.Refund(ctx, userID, 20000, "Refund for order 1 item")
		assert.NoError(t, err)

		bal, err = svc.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, int64(80000), bal)
	})

	t.Run("Deduct insufficient balance fails", func(t *testing.T) {
		err := svc.Deduct(ctx, userID, 200000, "Buying expensive medicine")
		assert.ErrorIs(t, err, ErrInsufficientBalance)
	})

	t.Run("ListTransactions details", func(t *testing.T) {
		list, total, err := svc.ListTransactions(ctx, userID, nil, 1, 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		assert.Len(t, list, total)
	})
}
