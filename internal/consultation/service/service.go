package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/dto"
)

var (
	ErrInvalidTransition = errors.New("invalid consultation status transition")
	ErrUnauthorized      = errors.New("unauthorized action on consultation")
)

type ConsultationService interface {
	CreateConsultation(ctx context.Context, appointmentID uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.ConsultationResponse, error)
	Start(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID) (*dto.ConsultationResponse, error)
	Complete(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID) (*dto.ConsultationResponse, error)
	UpdateNotes(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID, notes string) (*dto.ConsultationResponse, error)
	CancelConsultation(ctx context.Context, appointmentID uuid.UUID) error
}
