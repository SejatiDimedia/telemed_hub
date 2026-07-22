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

	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/model"
)

func setupTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	// Spin up PostgreSQL test container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
	)
	require.NoError(t, err)

	connURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create pgx connection pool with retry loop
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

	// Run migrations programmatically
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

func TestPostgresRepository_ConcurrencyBooking(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	// 1. Seed users, patient, doctor, and availability slot
	doctorUserID := uuid.New()
	patientUserID := uuid.New()
	doctorID := uuid.New()
	patientID := uuid.New()
	availabilityID := uuid.New()

	var doctorRoleID uuid.UUID
	var patientRoleID uuid.UUID

	err := db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'doctor'`).Scan(&doctorRoleID)
	require.NoError(t, err)
	err = db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'patient'`).Scan(&patientRoleID)
	require.NoError(t, err)

	// Seed User Doctor
	_, err = db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1, $2, $3, $4, $5)`, doctorUserID, "doc@test.com", "hash", "Doc Name", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, doctorUserID, doctorRoleID)
	require.NoError(t, err)

	// Seed User Patient
	_, err = db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1, $2, $3, $4, $5)`, patientUserID, "pat@test.com", "hash", "Pat Name", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, patientUserID, patientRoleID)
	require.NoError(t, err)

	// Seed Patient Profile
	_, err = db.Exec(ctx, `INSERT INTO patients (id, user_id) VALUES ($1, $2)`, patientID, patientUserID)
	require.NoError(t, err)

	// Seed Doctor Profile
	_, err = db.Exec(ctx, `INSERT INTO doctors (id, user_id, consultation_fee, is_credential_verified) VALUES ($1, $2, $3, $4)`,
		doctorID, doctorUserID, 100000, true)
	require.NoError(t, err)

	// Seed Availability Slot
	startTime := time.Now().Add(2 * time.Hour).UTC()
	endTime := startTime.Add(30 * time.Minute)
	_, err = db.Exec(ctx, `INSERT INTO doctor_availability (id, doctor_id, start_time, end_time, is_booked) VALUES ($1, $2, $3, $4, $5)`,
		availabilityID, doctorID, startTime, endTime, false)
	require.NoError(t, err)

	// 2. Perform concurrent booking attempts (10 goroutines)
	const numAttempts = 10
	var wg sync.WaitGroup
	errorsChan := make(chan error, numAttempts)

	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			apt := &model.Appointment{
				ID:             uuid.New(),
				PatientID:      patientID,
				DoctorID:       doctorID,
				AvailabilityID: availabilityID,
				Status:         "confirmed",
				ScheduledAt:    startTime,
			}
			err := repo.CreateWithLock(ctx, apt)
			errorsChan <- err
		}()
	}

	wg.Wait()
	close(errorsChan)

	// 3. Count success and conflict outcomes
	successCount := 0
	conflictCount := 0
	otherErrorCount := 0

	for err := range errorsChan {
		if err == nil {
			successCount++
		} else if err == ErrSlotAlreadyBooked {
			conflictCount++
		} else {
			t.Logf("unexpected error: %v", err)
			otherErrorCount++
		}
	}

	assert.Equal(t, 1, successCount, "exactly 1 booking should succeed")
	assert.Equal(t, numAttempts-1, conflictCount, "all other bookings must fail with slot already booked conflict")
	assert.Equal(t, 0, otherErrorCount, "no other errors should occur")

	// 4. Verify doctor availability is set to is_booked = true
	var isBooked bool
	err = db.QueryRow(ctx, "SELECT is_booked FROM doctor_availability WHERE id = $1", availabilityID).Scan(&isBooked)
	require.NoError(t, err)
	assert.True(t, isBooked)

	// 5. Verify exactly 1 appointment row exists
	var appointmentCount int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM appointments WHERE availability_id = $1", availabilityID).Scan(&appointmentCount)
	require.NoError(t, err)
	assert.Equal(t, 1, appointmentCount)
}
