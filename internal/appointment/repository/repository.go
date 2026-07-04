package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/model"
)

var (
	ErrAppointmentNotFound = errors.New("appointment not found")
	ErrAvailabilityNotFound = errors.New("availability slot not found")
	ErrSlotAlreadyBooked    = errors.New("availability slot already booked")
)

type AppointmentRepository interface {
	CreateWithLock(ctx context.Context, apt *model.Appointment) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Appointment, error)
	List(ctx context.Context, filter map[string]any) ([]*model.Appointment, error)
	UpdateStatusWithLock(ctx context.Context, id uuid.UUID, status string, cancelReason *string) error
	RescheduleWithLock(ctx context.Context, id uuid.UUID, newAvailabilityID uuid.UUID, scheduledAt time.Time) error
	GetAvailabilityByID(ctx context.Context, id uuid.UUID) (isBooked bool, doctorID uuid.UUID, startTime time.Time, endTime time.Time, err error)
}
