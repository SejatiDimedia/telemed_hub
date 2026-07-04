package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
)

// PatientService defines the patient profile use cases.
type PatientService interface {
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*dto.PatientResponse, error)
	GetProfileByID(ctx context.Context, id uuid.UUID) (*dto.PatientResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdatePatientRequest) (*dto.PatientResponse, error)
}
