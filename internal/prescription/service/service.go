package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/dto"
)

var (
	// ErrEmptyItems is returned when a prescription has no line-items.
	ErrEmptyItems = errors.New("prescription must have at least one item")

	// ErrInvalidConsultationStatus is returned when the consultation is not in a prescribable state.
	ErrInvalidConsultationStatus = errors.New("prescription can only be issued for in_progress or completed consultations")

	// ErrUnauthorized is returned when the caller is not the consultation's assigned doctor.
	ErrUnauthorized = errors.New("unauthorized: only the assigned doctor can issue a prescription")

	// ErrPrescriptionNotFound is returned when the prescription is not found or soft-deleted.
	ErrPrescriptionNotFound = errors.New("prescription not found")
)

// PrescriptionService defines the use cases for the prescription module.
type PrescriptionService interface {
	// Issue creates a new prescription for a completed or in-progress consultation.
	Issue(ctx context.Context, doctorUserID uuid.UUID, req dto.CreatePrescriptionRequest) (*dto.PrescriptionResponse, error)

	// GetByID returns a prescription if the caller is authorized to view it.
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.PrescriptionResponse, error)

	// List returns prescriptions scoped to the caller's role.
	List(ctx context.Context, userID uuid.UUID, roles []string) ([]*dto.PrescriptionResponse, error)
}
