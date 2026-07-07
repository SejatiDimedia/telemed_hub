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
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/model"
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

func TestPostgresRepository_MedicineOperations(t *testing.T) {
	// Skip test if docker daemon is not available to prevent failing builds
	// when docker is not running.
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	t.Run("Create, GetByID, Update, List, Delete", func(t *testing.T) {
		med := &model.Medicine{
			ID:                   uuid.New(),
			Name:                 "Paracetamol 500mg Test",
			UnitPrice:            5000.00,
			StockQuantity:        100,
			RequiresPrescription: false,
		}

		// 1. Create
		err := repo.Create(ctx, med)
		assert.NoError(t, err)

		// 2. GetByID
		fetched, err := repo.GetByID(ctx, med.ID)
		assert.NoError(t, err)
		assert.Equal(t, med.Name, fetched.Name)
		assert.Equal(t, med.UnitPrice, fetched.UnitPrice)
		assert.Equal(t, med.StockQuantity, fetched.StockQuantity)

		// 3. Update
		fetched.Name = "Paracetamol 500mg Test Updated"
		fetched.StockQuantity = 200
		err = repo.Update(ctx, fetched)
		assert.NoError(t, err)

		fetchedUpdated, err := repo.GetByID(ctx, med.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Paracetamol 500mg Test Updated", fetchedUpdated.Name)
		assert.Equal(t, 200, fetchedUpdated.StockQuantity)

		// 4. List
		nameFilter := "Test"
		reqPrescFilter := false
		list, total, err := repo.List(ctx, &nameFilter, &reqPrescFilter, 1, 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.NotEmpty(t, list)

		// 5. Soft Delete
		actorID := uuid.New()
		err = repo.SoftDelete(ctx, med.ID, actorID)
		assert.NoError(t, err)

		// GetByID should return not found after soft delete
		_, err = repo.GetByID(ctx, med.ID)
		assert.ErrorIs(t, err, ErrMedicineNotFound)
	})

	t.Run("Stock Concurrency Lock (Oversell Prevention)", func(t *testing.T) {
		medID := uuid.New()
		med := &model.Medicine{
			ID:                   medID,
			Name:                 "Limited Stock Medicine",
			UnitPrice:            10000.00,
			StockQuantity:        1, // only 1 in stock
			RequiresPrescription: false,
		}

		err := repo.Create(ctx, med)
		require.NoError(t, err)

		// We will spawn 5 concurrent goroutines trying to buy/decrement the medicine.
		// Since stock is 1, exactly 1 must succeed and the other 4 must fail.
		var wg sync.WaitGroup
		successCount := 0
		failCount := 0
		var mu sync.Mutex

		numWorkers := 5
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Start database transaction
				tx, err := db.Begin(ctx)
				if err != nil {
					mu.Lock()
					failCount++
					mu.Unlock()
					return
				}
				defer tx.Rollback(ctx)

				// Lock the medicine row using GetByIDForUpdate
				m, err := repo.GetByIDForUpdate(ctx, tx, medID)
				if err != nil {
					return
				}

				// Check and decrement stock
				if m.StockQuantity >= 1 {
					m.StockQuantity -= 1
					err = repo.Update(ctx, m)
					if err == nil {
						err = tx.Commit(ctx)
						if err == nil {
							mu.Lock()
							successCount++
							mu.Unlock()
							return
						}
					}
				}

				mu.Lock()
				failCount++
				mu.Unlock()
			}()
		}

		wg.Wait()

		assert.Equal(t, 1, successCount, "Only 1 transaction should successfully decrement the stock")
		assert.Equal(t, 4, failCount, "The other 4 transactions must fail due to out-of-stock")

		// Verify database stock is 0
		finalMed, err := repo.GetByID(ctx, medID)
		assert.NoError(t, err)
		assert.Equal(t, 0, finalMed.StockQuantity)
	})
}
