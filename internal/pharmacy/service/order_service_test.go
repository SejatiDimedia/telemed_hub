package service

import (
	"context"
	"errors"
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
	inventoryRepository "github.com/timurdianradhasejati/telemed_hub/internal/inventory/repository"
	inventoryService "github.com/timurdianradhasejati/telemed_hub/internal/inventory/service"
	patientRepository "github.com/timurdianradhasejati/telemed_hub/internal/patient/repository"
	patientService "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	prescriptionRepository "github.com/timurdianradhasejati/telemed_hub/internal/prescription/repository"
	prescriptionService "github.com/timurdianradhasejati/telemed_hub/internal/prescription/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/repository"
	walletRepository "github.com/timurdianradhasejati/telemed_hub/internal/wallet/repository"
	walletService "github.com/timurdianradhasejati/telemed_hub/internal/wallet/service"
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

func seedCompleteRequiredData(t *testing.T, ctx context.Context, db *pgxpool.Pool) (
	patientUserID, doctorUserID, consultationID, prescriptionID, medicineID uuid.UUID,
) {
	patientUserID = uuid.New()
	doctorUserID = uuid.New()
	doctorID := uuid.New()
	patientID := uuid.New()
	availabilityID := uuid.New()
	consultationID = uuid.New()
	prescriptionID = uuid.New()
	medicineID = uuid.New()

	var doctorRoleID, patientRoleID uuid.UUID
	require.NoError(t, db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'doctor'`).Scan(&doctorRoleID))
	require.NoError(t, db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'patient'`).Scan(&patientRoleID))

	_, err := db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1,$2,$3,$4,$5)`,
		doctorUserID, "doc@ordersvc.com", "hash", "Doc Svc", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2)`, doctorUserID, doctorRoleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1,$2,$3,$4,$5)`,
		patientUserID, "pat@ordersvc.com", "hash", "Pat Svc", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2)`, patientUserID, patientRoleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO patients (id, user_id) VALUES ($1,$2)`, patientID, patientUserID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO doctors (id, user_id, consultation_fee, is_credential_verified) VALUES ($1,$2,$3,$4)`,
		doctorID, doctorUserID, 150000, true)
	require.NoError(t, err)

	startTime := time.Now().Add(5 * time.Hour).UTC()
	endTime := startTime.Add(30 * time.Minute)
	_, err = db.Exec(ctx, `INSERT INTO doctor_availability (id, doctor_id, start_time, end_time, is_booked) VALUES ($1,$2,$3,$4,$5)`,
		availabilityID, doctorID, startTime, endTime, true)
	require.NoError(t, err)

	appointmentID := uuid.New()
	_, err = db.Exec(ctx, `INSERT INTO appointments (id, patient_id, doctor_id, availability_id, status, scheduled_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		appointmentID, patientID, doctorID, availabilityID, "confirmed", startTime)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO consultations (id, appointment_id, status) VALUES ($1,$2,$3)`,
		consultationID, appointmentID, "completed")
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO prescriptions (id, consultation_id, patient_id, doctor_id, status) VALUES ($1,$2,$3,$4,$5)`,
		prescriptionID, consultationID, patientID, doctorID, "active")
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO medicines (id, name, unit_price, stock_quantity, requires_prescription) VALUES ($1,$2,$3,$4,$5)`,
		medicineID, "Paracetamol 500mg Svc Test", 5000, 1, true) // Only 1 stock
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO prescription_items (id, prescription_id, medicine_id, dosage, quantity) VALUES ($1,$2,$3,$4,$5)`,
		uuid.New(), prescriptionID, medicineID, "1 tablet daily", 1)
	require.NoError(t, err)

	return patientUserID, doctorUserID, consultationID, prescriptionID, medicineID
}

func TestOrderService_Operations(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()

	// Instantiate all required sub-services
	invRepo := inventoryRepository.NewPostgresRepository(db)
	invSvc := inventoryService.NewInventoryService(invRepo)

	patRepo := patientRepository.NewPostgresRepository(db)
	patSvc := patientService.NewPatientService(patRepo)

	presRepo := prescriptionRepository.NewPostgresRepository(db)
	presSvc := prescriptionService.NewPrescriptionService(presRepo, nil, nil, patSvc)

	walRepo := walletRepository.NewPostgresRepository(db)
	walSvc := walletService.NewWalletService(walRepo, db)

	orderRepo := repository.NewPostgresRepository(db)
	orderSvc := NewOrderService(orderRepo, db, presSvc, invSvc, patSvc, walSvc, nil)

	// Seed dependencies
	patientUserID, _, _, prescriptionID, medicineID := seedCompleteRequiredData(t, ctx, db)

	// Setup wallet for the patient user
	key := "seed-topup"
	_, err := walSvc.TopUp(ctx, patientUserID, 10000.0, &key)
	require.NoError(t, err)

	t.Run("Create order fails if insufficient balance", func(t *testing.T) {
		// Seed prescription with high cost medicine
		expensiveMedID := uuid.New()
		_, err = db.Exec(ctx, `INSERT INTO medicines (id, name, unit_price, stock_quantity, requires_prescription) VALUES ($1,$2,$3,$4,$5)`,
			expensiveMedID, "Super Expensive Drug", 20000, 10, true)
		require.NoError(t, err)

		expensivePrescID := uuid.New()
		_, err = db.Exec(ctx, `INSERT INTO prescriptions (id, consultation_id, patient_id, doctor_id, status) VALUES ($1,(SELECT id FROM consultations LIMIT 1),(SELECT id FROM patients LIMIT 1),(SELECT id FROM doctors LIMIT 1),'active')`,
			expensivePrescID)
		require.NoError(t, err)

		_, err = db.Exec(ctx, `INSERT INTO prescription_items (id, prescription_id, medicine_id, dosage, quantity) VALUES ($1,$2,$3,$4,$5)`,
			uuid.New(), expensivePrescID, expensiveMedID, "1 daily", 1)
		require.NoError(t, err)

		req := dto.CreateOrderRequest{PrescriptionID: expensivePrescID.String()}
		_, err = orderSvc.Create(ctx, patientUserID, req, nil)
		assert.ErrorIs(t, err, ErrInsufficientBalance)
	})

	t.Run("Concurrent checkout stock oversell prevention", func(t *testing.T) {
		req := dto.CreateOrderRequest{PrescriptionID: prescriptionID.String()}

		// We will trigger 2 concurrent order checkouts using the prescription containing 1 stock Paracetamol.
		// One must succeed, the other must fail with ErrOutOfStock.
		var wg sync.WaitGroup
		successCount := 0
		failCount := 0
		var mu sync.Mutex

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := orderSvc.Create(ctx, patientUserID, req, nil)
				mu.Lock()
				if err == nil {
					successCount++
				} else if errors.Is(err, ErrOutOfStock) {
					failCount++
				}
				mu.Unlock()
			}()
		}
		wg.Wait()

		assert.Equal(t, 1, successCount, "Exactly 1 order checkout should succeed")
		assert.Equal(t, 1, failCount, "Exactly 1 order checkout should fail with out-of-stock")

		// Verify medicine stock is 0
		med, err := invSvc.GetByID(ctx, medicineID)
		assert.NoError(t, err)
		assert.Equal(t, 0, med.StockQuantity)
	})

	t.Run("Cancel order releases stock and refunds wallet", func(t *testing.T) {
		// List patient's orders to find the successful order
		orders, err := orderSvc.List(ctx, patientUserID, []string{"patient"}, "")
		require.NoError(t, err)
		require.NotEmpty(t, orders)

		ordID, _ := uuid.Parse(orders[0].ID)

		// Get patient wallet balance before cancel (initial 10000 - 5000 = 5000)
		balBefore, err := walSvc.GetBalance(ctx, patientUserID)
		require.NoError(t, err)
		assert.Equal(t, int64(5000), balBefore)

		// Cancel order
		err = orderSvc.Cancel(ctx, patientUserID, ordID)
		assert.NoError(t, err)

		// Verify stock is restored to 1
		med, err := invSvc.GetByID(ctx, medicineID)
		assert.NoError(t, err)
		assert.Equal(t, 1, med.StockQuantity)

		// Verify wallet is refunded to 10000
		balAfter, err := walSvc.GetBalance(ctx, patientUserID)
		assert.NoError(t, err)
		assert.Equal(t, int64(10000), balAfter)
	})
}
