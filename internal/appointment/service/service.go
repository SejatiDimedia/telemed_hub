package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/dto"
)

var (
	ErrProfileIncomplete          = errors.New("patient profile is incomplete (must provide date_of_birth and gender)")
	ErrDoctorNotVerified          = errors.New("doctor is not verified by admin")
	ErrCancellationCutoffExpired  = errors.New("cannot cancel or reschedule within cutoff window of appointment")
	ErrInsufficientBalance        = errors.New("insufficient wallet balance for consultation fee")
	ErrUnauthorized               = errors.New("unauthorized access to appointment")
	ErrAppointmentAlreadyCancelled = errors.New("appointment is already cancelled")
)

type AppointmentService interface {
	Book(ctx context.Context, patientUserID uuid.UUID, req dto.CreateAppointmentRequest) (*dto.AppointmentResponse, error)
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.AppointmentResponse, error)
	List(ctx context.Context, userID uuid.UUID, roles []string, statusFilter string) ([]*dto.AppointmentResponse, error)
	Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CancelAppointmentRequest) error
	Reschedule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.RescheduleAppointmentRequest) (*dto.AppointmentResponse, error)
}
