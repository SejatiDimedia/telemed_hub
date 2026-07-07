package repository

import (
	"context"
	"path/filepath"
	"sync"
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
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/model"
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
		userID, "walletpat@test.com", "hash", "Pat Wallet", "active")
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2)`, userID, roleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO patients (id, user_id) VALUES ($1,$2)`, patientID, userID)
	require.NoError(t, err)

	return userID, patientID
}

func TestPostgresRepository_WalletOperations(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	userID, patientID := seedPatientUser(t, ctx, db)

	t.Run("GetOrCreateWallet and GetWalletByUserID", func(t *testing.T) {
		w, err := repo.GetOrCreateWallet(ctx, nil, patientID)
		assert.NoError(t, err)
		assert.NotNil(t, w)
		assert.Equal(t, patientID, w.PatientID)
		assert.Equal(t, 0.0, w.Balance)

		// Get wallet by user ID should fetch the same wallet
		w2, err := repo.GetWalletByUserID(ctx, nil, userID)
		assert.NoError(t, err)
		assert.Equal(t, w.ID, w2.ID)
	})

	t.Run("UpdateWalletBalance & CreateTransaction & LedgerConsistency & GetTransactionByIdempotencyKey", func(t *testing.T) {
		w, err := repo.GetWalletByUserID(ctx, nil, userID)
		require.NoError(t, err)

		tx, err := db.Begin(ctx)
		require.NoError(t, err)

		err = repo.UpdateWalletBalance(ctx, tx, w.ID, 150000.0)
		assert.NoError(t, err)

		key := "idem-key-1"
		txRecord := &model.WalletTransaction{
			WalletID:       w.ID,
			Type:           "top_up",
			Amount:         150000.0,
			BalanceAfter:   150000.0,
			IdempotencyKey: &key,
		}

		err = repo.CreateTransaction(ctx, tx, txRecord)
		assert.NoError(t, err)

		err = tx.Commit(ctx)
		assert.NoError(t, err)

		// Get wallet to verify new balance
		wUpdated, err := repo.GetWalletByUserID(ctx, nil, userID)
		assert.NoError(t, err)
		assert.Equal(t, 150000.0, wUpdated.Balance)

		// Verify idempotency check finds it
		existing, err := repo.GetTransactionByIdempotencyKey(ctx, nil, key)
		assert.NoError(t, err)
		assert.NotNil(t, existing)
		assert.Equal(t, 150000.0, existing.Amount)

		// Verify ledger consistency check matches balance
		calculated, err := repo.VerifyLedgerConsistency(ctx, w.ID)
		assert.NoError(t, err)
		assert.Equal(t, 150000.0, calculated)
	})

	t.Run("ListTransactions", func(t *testing.T) {
		w, err := repo.GetWalletByUserID(ctx, nil, userID)
		require.NoError(t, err)

		list, total, err := repo.ListTransactions(ctx, w.ID, nil, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, list, 1)
	})

	t.Run("Concurrent Wallet Operations (Atomicity Lock)", func(t *testing.T) {
		_, err := repo.GetWalletByUserID(ctx, nil, userID)
		require.NoError(t, err)

		// Spawn 10 concurrent goroutines doing top-up of 1000 each using tx row locks
		var wg sync.WaitGroup
		numWorkers := 10
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tx, err := db.Begin(ctx)
				if err != nil {
					return
				}
				defer tx.Rollback(ctx)

				lockedWallet, err := repo.GetWalletByUserIDForUpdate(ctx, tx, userID)
				if err != nil {
					return
				}

				newBal := lockedWallet.Balance + 1000.0
				err = repo.UpdateWalletBalance(ctx, tx, lockedWallet.ID, newBal)
				if err != nil {
					return
				}

				txRec := &model.WalletTransaction{
					WalletID:     lockedWallet.ID,
					Type:         "top_up",
					Amount:       1000.0,
					BalanceAfter: newBal,
				}
				err = repo.CreateTransaction(ctx, tx, txRec)
				if err != nil {
					return
				}

				_ = tx.Commit(ctx)
			}()
		}
		wg.Wait()

		// Final balance should be previous balance (150,000) + 10 * 1000 = 160,000
		finalWallet, err := repo.GetWalletByUserID(ctx, nil, userID)
		assert.NoError(t, err)
		assert.Equal(t, 160000.0, finalWallet.Balance)
	})
}
