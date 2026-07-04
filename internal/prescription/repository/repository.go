package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/model"
)

var (
	ErrPrescriptionNotFound = errors.New("prescription not found")
	ErrMedicineNotFound     = errors.New("one or more medicines not found")
)

// PrescriptionRepository defines persistence operations for prescriptions.
type PrescriptionRepository interface {
	// Create persists the prescription header and all line-items atomically.
	Create(ctx context.Context, pres *model.Prescription) error

	// GetByID fetches a prescription with its items (including medicine name).
	// Returns ErrPrescriptionNotFound if not found or soft-deleted.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Prescription, error)

	// ListByPatientID returns all active prescriptions for a patient.
	ListByPatientID(ctx context.Context, patientID uuid.UUID) ([]*model.Prescription, error)

	// ListByDoctorID returns all prescriptions issued by a doctor.
	ListByDoctorID(ctx context.Context, doctorID uuid.UUID) ([]*model.Prescription, error)
}
