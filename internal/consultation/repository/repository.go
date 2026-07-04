package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/model"
)

var (
	ErrConsultationNotFound = errors.New("consultation not found")
)

type ConsultationRepository interface {
	Create(ctx context.Context, cons *model.Consultation) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Consultation, error)
	GetByAppointmentID(ctx context.Context, aptID uuid.UUID) (*model.Consultation, error)
	Update(ctx context.Context, cons *model.Consultation) error
}
