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

	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/model"
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

func TestPostgresRepository_ConsultationOperations(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	// 1. Seed users, patient, doctor, availability slot, and appointment
	doctorUserID := uuid.New()
	patientUserID := uuid.New()
	doctorID := uuid.New()
	patientID := uuid.New()
	availabilityID := uuid.New()
	appointmentID := uuid.New()

	var doctorRoleID uuid.UUID
	var patientRoleID uuid.UUID
	err := db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'doctor'`).Scan(&doctorRoleID)
	require.NoError(t, err)
	err = db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'patient'`).Scan(&patientRoleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1, $2, $3, $4, $5)`, doctorUserID, "doc@test.com", "hash", "Doc Name", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, doctorUserID, doctorRoleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1, $2, $3, $4, $5)`, patientUserID, "pat@test.com", "hash", "Pat Name", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, patientUserID, patientRoleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO patients (id, user_id) VALUES ($1, $2)`, patientID, patientUserID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO doctors (id, user_id, consultation_fee, is_credential_verified) VALUES ($1, $2, $3, $4)`,
		doctorID, doctorUserID, 100000, true)
	require.NoError(t, err)

	startTime := time.Now().Add(2 * time.Hour).UTC()
	endTime := startTime.Add(30 * time.Minute)
	_, err = db.Exec(ctx, `INSERT INTO doctor_availability (id, doctor_id, start_time, end_time, is_booked) VALUES ($1, $2, $3, $4, $5)`,
		availabilityID, doctorID, startTime, endTime, true)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO appointments (id, patient_id, doctor_id, availability_id, status, scheduled_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		appointmentID, patientID, doctorID, availabilityID, "confirmed", startTime)
	require.NoError(t, err)

	// 2. Test Create Consultation
	cons := &model.Consultation{
		ID:            uuid.New(),
		AppointmentID: appointmentID,
		Status:        "scheduled",
	}
	err = repo.Create(ctx, cons)
	assert.NoError(t, err)

	// 3. Test GetByID
	fetched, err := repo.GetByID(ctx, cons.ID)
	assert.NoError(t, err)
	assert.Equal(t, cons.ID, fetched.ID)
	assert.Equal(t, "scheduled", fetched.Status)

	// 4. Test GetByAppointmentID
	fetchedByApt, err := repo.GetByAppointmentID(ctx, appointmentID)
	assert.NoError(t, err)
	assert.Equal(t, cons.ID, fetchedByApt.ID)

	// 5. Test Update
	now := time.Now().UTC()
	notes := "Patient is doing fine"
	fetched.Status = "in_progress"
	fetched.StartedAt = &now
	fetched.Notes = &notes

	err = repo.Update(ctx, fetched)
	assert.NoError(t, err)

	fetchedUpdated, err := repo.GetByID(ctx, cons.ID)
	assert.NoError(t, err)
	assert.Equal(t, "in_progress", fetchedUpdated.Status)
	assert.Equal(t, notes, *fetchedUpdated.Notes)
	assert.NotNil(t, fetchedUpdated.StartedAt)
}
