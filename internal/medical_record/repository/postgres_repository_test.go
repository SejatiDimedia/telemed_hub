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
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/model"
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

func seedTreatmentEntities(t *testing.T, ctx context.Context, db *pgxpool.Pool) (doctorUserID, patientID, consultationID uuid.UUID) {
	doctorUserID = uuid.New()
	patientUserID := uuid.New()
	doctorID := uuid.New()
	patientID = uuid.New()
	availabilityID := uuid.New()
	appointmentID := uuid.New()
	consultationID = uuid.New()

	var doctorRoleID, patientRoleID uuid.UUID
	require.NoError(t, db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'doctor'`).Scan(&doctorRoleID))
	require.NoError(t, db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'patient'`).Scan(&patientRoleID))

	_, err := db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1,$2,$3,$4,$5)`,
		doctorUserID, "mrdoc@test.com", "hash", "MR Doctor", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2)`, doctorUserID, doctorRoleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1,$2,$3,$4,$5)`,
		patientUserID, "mrpat@test.com", "hash", "MR Patient", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2)`, patientUserID, patientRoleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO patients (id, user_id) VALUES ($1,$2)`, patientID, patientUserID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO doctors (id, user_id, consultation_fee, is_credential_verified) VALUES ($1,$2,$3,$4)`,
		doctorID, doctorUserID, 150000, true)
	require.NoError(t, err)

	startTime := time.Now().Add(1 * time.Hour).UTC()
	endTime := startTime.Add(30 * time.Minute)
	_, err = db.Exec(ctx, `INSERT INTO doctor_availability (id, doctor_id, start_time, end_time, is_booked) VALUES ($1,$2,$3,$4,$5)`,
		availabilityID, doctorID, startTime, endTime, true)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO appointments (id, patient_id, doctor_id, availability_id, status, scheduled_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		appointmentID, patientID, doctorID, availabilityID, "completed", startTime)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO consultations (id, appointment_id, status) VALUES ($1,$2,$3)`,
		consultationID, appointmentID, "completed")
	require.NoError(t, err)

	return doctorUserID, patientID, consultationID
}

func TestPostgresRepository_MedicalRecordOperations(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	doctorUserID, patientID, consultationID := seedTreatmentEntities(t, ctx, db)

	t.Run("Create & GetByID & HasTreatmentRelationship", func(t *testing.T) {
		// Verify treatment relationship
		hasRelationship, err := repo.HasTreatmentRelationship(ctx, doctorUserID, patientID)
		assert.NoError(t, err)
		assert.True(t, hasRelationship)

		// Verification for non-treated patient
		otherPatientID := uuid.New()
		_, err = db.Exec(ctx, `INSERT INTO patients (id, user_id) VALUES ($1, (SELECT id FROM users LIMIT 1))`, otherPatientID)
		require.NoError(t, err)

		hasNoRelationship, err := repo.HasTreatmentRelationship(ctx, doctorUserID, otherPatientID)
		assert.NoError(t, err)
		assert.False(t, hasNoRelationship)

		// Create record
		recID := uuid.New()
		rec := &model.MedicalRecord{
			ID:             recID,
			PatientID:      patientID,
			ConsultationID: &consultationID,
			RecordType:     "diagnosis",
			Content:        "Patient exhibits mild symptoms of flu.",
			CreatedBy:      &doctorUserID,
		}

		err = repo.Create(ctx, rec)
		assert.NoError(t, err)

		// Fetch and verify
		fetched, err := repo.GetByID(ctx, recID)
		assert.NoError(t, err)
		assert.Equal(t, recID, fetched.ID)
		assert.Equal(t, "diagnosis", fetched.RecordType)
		assert.Equal(t, "Patient exhibits mild symptoms of flu.", fetched.Content)
		assert.Equal(t, doctorUserID, *fetched.CreatedBy)
	})

	t.Run("List and Update & Soft Delete", func(t *testing.T) {
		recID := uuid.New()
		rec := &model.MedicalRecord{
			ID:         recID,
			PatientID:  patientID,
			RecordType: "allergy",
			Content:    "Allergic to Penicillin",
			CreatedBy:  &doctorUserID,
		}

		err := repo.Create(ctx, rec)
		require.NoError(t, err)

		// Update
		rec.Content = "Allergic to Penicillin and Amoxicillin"
		err = repo.Update(ctx, rec)
		assert.NoError(t, err)

		// Verify update saved
		fetched, err := repo.GetByID(ctx, recID)
		assert.NoError(t, err)
		assert.Equal(t, "Allergic to Penicillin and Amoxicillin", fetched.Content)

		// List filtered by type
		patStr := patientID.String()
		typeStr := "allergy"
		list, err := repo.List(ctx, dto.ListMedicalRecordsFilter{
			PatientID:  &patStr,
			RecordType: &typeStr,
		})
		assert.NoError(t, err)
		assert.Len(t, list, 1)

		// Soft Delete
		err = repo.SoftDelete(ctx, recID, doctorUserID)
		assert.NoError(t, err)

		// GetByID after soft delete should fail
		_, err = repo.GetByID(ctx, recID)
		assert.ErrorIs(t, err, ErrMedicalRecordNotFound)
	})
}
