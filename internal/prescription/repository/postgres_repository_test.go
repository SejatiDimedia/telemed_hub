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

	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/model"
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

// seedRequiredData inserts the minimum set of rows required by the FK constraints
// of prescriptions (users → patients/doctors → doctor_availability → appointments → consultations).
func seedRequiredData(t *testing.T, ctx context.Context, db *pgxpool.Pool) (
	doctorID, patientID, consultationID uuid.UUID,
) {
	t.Helper()

	doctorUserID := uuid.New()
	patientUserID := uuid.New()
	doctorID = uuid.New()
	patientID = uuid.New()
	availabilityID := uuid.New()
	appointmentID := uuid.New()
	consultationID = uuid.New()

	var doctorRoleID, patientRoleID uuid.UUID
	require.NoError(t, db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'doctor'`).Scan(&doctorRoleID))
	require.NoError(t, db.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'patient'`).Scan(&patientRoleID))

	_, err := db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1,$2,$3,$4,$5)`,
		doctorUserID, "doc@presctest.com", "hash", "Doc Name", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2)`, doctorUserID, doctorRoleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, status) VALUES ($1,$2,$3,$4,$5)`,
		patientUserID, "pat@presctest.com", "hash", "Pat Name", "active")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2)`, patientUserID, patientRoleID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO patients (id, user_id) VALUES ($1,$2)`, patientID, patientUserID)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO doctors (id, user_id, consultation_fee, is_credential_verified) VALUES ($1,$2,$3,$4)`,
		doctorID, doctorUserID, 100000, true)
	require.NoError(t, err)

	startTime := time.Now().Add(2 * time.Hour).UTC()
	endTime := startTime.Add(30 * time.Minute)
	_, err = db.Exec(ctx, `INSERT INTO doctor_availability (id, doctor_id, start_time, end_time, is_booked) VALUES ($1,$2,$3,$4,$5)`,
		availabilityID, doctorID, startTime, endTime, true)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO appointments (id, patient_id, doctor_id, availability_id, status, scheduled_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		appointmentID, patientID, doctorID, availabilityID, "confirmed", startTime)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `INSERT INTO consultations (id, appointment_id, status) VALUES ($1,$2,$3)`,
		consultationID, appointmentID, "in_progress")
	require.NoError(t, err)

	return doctorID, patientID, consultationID
}

func TestPostgresRepository_PrescriptionOperations(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	// Seed FK dependencies
	doctorID, patientID, consultationID := seedRequiredData(t, ctx, db)

	// Seed a medicine to reference
	medicineID := uuid.New()
	_, err := db.Exec(ctx, `INSERT INTO medicines (id, name, unit_price, stock_quantity, requires_prescription) VALUES ($1,$2,$3,$4,$5)`,
		medicineID, "Test Medicine", 10000, 100, true)
	require.NoError(t, err)

	t.Run("Create and GetByID", func(t *testing.T) {
		instructions := "Take with food"
		pres := &model.Prescription{
			ConsultationID: consultationID,
			PatientID:      patientID,
			DoctorID:       doctorID,
			Status:         "active",
			Items: []model.PrescriptionItem{
				{
					MedicineID:   medicineID,
					Dosage:       "500mg twice daily",
					Quantity:     10,
					Instructions: &instructions,
				},
			},
		}

		err := repo.Create(ctx, pres)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, pres.ID)

		// Fetch the saved prescription
		fetched, err := repo.GetByID(ctx, pres.ID)
		assert.NoError(t, err)
		assert.Equal(t, pres.ID, fetched.ID)
		assert.Equal(t, "active", fetched.Status)
		assert.Equal(t, consultationID, fetched.ConsultationID)
		assert.Equal(t, patientID, fetched.PatientID)
		assert.Equal(t, doctorID, fetched.DoctorID)

		// Verify line-items were saved and medicine name was joined
		assert.Len(t, fetched.Items, 1)
		assert.Equal(t, medicineID, fetched.Items[0].MedicineID)
		assert.Equal(t, "Test Medicine", fetched.Items[0].MedicineName)
		assert.Equal(t, 10, fetched.Items[0].Quantity)
		assert.Equal(t, "Take with food", *fetched.Items[0].Instructions)
	})

	t.Run("GetByID returns not found for unknown ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		assert.ErrorIs(t, err, ErrPrescriptionNotFound)
	})

	t.Run("ListByPatientID returns correct prescriptions", func(t *testing.T) {
		// Create a second prescription for the same patient
		pres2 := &model.Prescription{
			ConsultationID: consultationID,
			PatientID:      patientID,
			DoctorID:       doctorID,
			Status:         "active",
			Items: []model.PrescriptionItem{
				{MedicineID: medicineID, Dosage: "10mg daily", Quantity: 5},
			},
		}
		require.NoError(t, repo.Create(ctx, pres2))

		records, err := repo.ListByPatientID(ctx, patientID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 2, "expected at least 2 prescriptions for patient")
	})

	t.Run("ListByDoctorID returns correct prescriptions", func(t *testing.T) {
		records, err := repo.ListByDoctorID(ctx, doctorID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 1, "expected at least one prescription for doctor")
	})
}
