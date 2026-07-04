package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/model"
)

var (
	ErrPatientNotFound = errors.New("patient not found")
)

// PatientRepository defines the interface for Patient data operations.
type PatientRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Patient, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Patient, error)
	Update(ctx context.Context, patient *model.Patient) error
}
