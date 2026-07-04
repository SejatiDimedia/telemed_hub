package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/model"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create inserts the prescription header and all line-items in a single transaction.
func (r *PostgresRepository) Create(ctx context.Context, pres *model.Prescription) error {
	if pres.ID == uuid.Nil {
		pres.ID = uuid.New()
	}
	now := time.Now().UTC()
	pres.IssuedAt = now
	pres.CreatedAt = now
	pres.UpdatedAt = now

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Insert prescription header
	_, err = tx.Exec(ctx, `
		INSERT INTO prescriptions
			(id, consultation_id, patient_id, doctor_id, issued_at, status, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		pres.ID, pres.ConsultationID, pres.PatientID, pres.DoctorID,
		pres.IssuedAt, pres.Status, pres.CreatedAt, pres.UpdatedAt, pres.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to insert prescription: %w", err)
	}

	// Insert each line-item
	for i := range pres.Items {
		item := &pres.Items[i]
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
		item.PrescriptionID = pres.ID
		item.CreatedAt = now
		item.UpdatedAt = now

		_, err = tx.Exec(ctx, `
			INSERT INTO prescription_items
				(id, prescription_id, medicine_id, dosage, quantity, instructions, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			item.ID, item.PrescriptionID, item.MedicineID,
			item.Dosage, item.Quantity, item.Instructions,
			item.CreatedAt, item.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert prescription item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetByID fetches a prescription header + items (with medicine name JOIN).
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Prescription, error) {
	// Fetch header
	var pres model.Prescription
	err := r.db.QueryRow(ctx, `
		SELECT id, consultation_id, patient_id, doctor_id, issued_at, status, created_at, updated_at
		FROM prescriptions
		WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(
		&pres.ID, &pres.ConsultationID, &pres.PatientID, &pres.DoctorID,
		&pres.IssuedAt, &pres.Status, &pres.CreatedAt, &pres.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPrescriptionNotFound
		}
		return nil, fmt.Errorf("failed to fetch prescription: %w", err)
	}

	// Fetch items with medicine name
	rows, err := r.db.Query(ctx, `
		SELECT pi.id, pi.prescription_id, pi.medicine_id, m.name,
		       pi.dosage, pi.quantity, pi.instructions, pi.created_at, pi.updated_at
		FROM prescription_items pi
		JOIN medicines m ON m.id = pi.medicine_id
		WHERE pi.prescription_id = $1
		ORDER BY pi.created_at ASC`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prescription items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.PrescriptionItem
		if err := rows.Scan(
			&item.ID, &item.PrescriptionID, &item.MedicineID, &item.MedicineName,
			&item.Dosage, &item.Quantity, &item.Instructions,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan prescription item: %w", err)
		}
		pres.Items = append(pres.Items, item)
	}

	return &pres, nil
}

// ListByPatientID returns all non-deleted prescriptions for a given patient.
func (r *PostgresRepository) ListByPatientID(ctx context.Context, patientID uuid.UUID) ([]*model.Prescription, error) {
	return r.listByField(ctx, "patient_id", patientID)
}

// ListByDoctorID returns all non-deleted prescriptions issued by a doctor.
func (r *PostgresRepository) ListByDoctorID(ctx context.Context, doctorID uuid.UUID) ([]*model.Prescription, error) {
	return r.listByField(ctx, "doctor_id", doctorID)
}

// listByField is a shared helper for listing prescriptions by a specific field.
func (r *PostgresRepository) listByField(ctx context.Context, field string, id uuid.UUID) ([]*model.Prescription, error) {
	query := fmt.Sprintf(`
		SELECT id, consultation_id, patient_id, doctor_id, issued_at, status, created_at, updated_at
		FROM prescriptions
		WHERE %s = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`, field)

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to list prescriptions: %w", err)
	}
	defer rows.Close()

	var result []*model.Prescription
	for rows.Next() {
		var pres model.Prescription
		if err := rows.Scan(
			&pres.ID, &pres.ConsultationID, &pres.PatientID, &pres.DoctorID,
			&pres.IssuedAt, &pres.Status, &pres.CreatedAt, &pres.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan prescription: %w", err)
		}
		result = append(result, &pres)
	}

	return result, nil
}
