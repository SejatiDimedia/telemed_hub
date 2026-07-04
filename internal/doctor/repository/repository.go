package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
)

var (
	ErrDoctorNotFound       = errors.New("doctor not found")
	ErrAvailabilityNotFound = errors.New("availability slot not found")
	ErrSlotBooked           = errors.New("availability slot is already booked")
)

// DoctorRepository defines data operations for Doctor profiles.
type DoctorRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Doctor, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Doctor, error)
	Update(ctx context.Context, doctor *model.Doctor) error
	Verify(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, specialty *string, onlyVerified bool, sortBy string, order string, offset int, limit int) ([]*model.Doctor, int, error)

	CreateAvailability(ctx context.Context, slot *model.Availability) error
	DeleteAvailability(ctx context.Context, doctorID uuid.UUID, slotID uuid.UUID) error
	GetAvailabilityByID(ctx context.Context, slotID uuid.UUID) (*model.Availability, error)
	ListAvailability(ctx context.Context, doctorID uuid.UUID, startTime time.Time, endTime time.Time, isBooked *bool) ([]*model.Availability, error)
	CheckOverlappingSlot(ctx context.Context, doctorID uuid.UUID, startTime time.Time, endTime time.Time) (bool, error)
}
