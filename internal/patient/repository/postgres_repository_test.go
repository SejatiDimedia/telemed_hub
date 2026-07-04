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

func TestPostgresRepository_PatientOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	// 1. Setup User and Patient Profile rows
	userID := uuid.New()
	patientID := uuid.New()

	queryUser := `
		INSERT INTO users (id, email, password_hash, full_name, is_verified, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := db.Exec(ctx, queryUser, userID, "rina@test.com", "hash", "Rina Patient", true, "active", time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)

	queryPatient := `
		INSERT INTO patients (id, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)`

	_, err = db.Exec(ctx, queryPatient, patientID, userID, time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)

	// 2. Fetch by User ID
	patient, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, patientID, patient.ID)
	assert.Equal(t, userID, patient.UserID)
	assert.Equal(t, "rina@test.com", patient.Email)
	assert.Equal(t, "Rina Patient", patient.FullName)
	assert.Nil(t, patient.PhoneNumber)

	// 3. Fetch by Patient ID
	patient2, err := repo.GetByID(ctx, patientID)
	require.NoError(t, err)
	assert.Equal(t, userID, patient2.UserID)

	// 4. Update Profile (Join Transactional)
	phone := "+6281234567890"
	dob := time.Date(1995, 4, 12, 0, 0, 0, 0, time.UTC)
	gender := "female"
	bloodType := "O+"

	patient.PhoneNumber = &phone
	patient.DateOfBirth = &dob
	patient.Gender = &gender
	patient.BloodType = &bloodType

	err = repo.Update(ctx, patient)
	require.NoError(t, err)

	// Fetch again to verify updates persisted
	updatedPatient, err := repo.GetByID(ctx, patientID)
	require.NoError(t, err)
	assert.Equal(t, "+6281234567890", *updatedPatient.PhoneNumber)
	assert.Equal(t, "female", *updatedPatient.Gender)
	assert.Equal(t, "O+", *updatedPatient.BloodType)
	assert.Equal(t, dob.Year(), updatedPatient.DateOfBirth.Year())
}
