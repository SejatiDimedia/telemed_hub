package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
)

var (
	ErrDoctorNotFound = errors.New("doctor not found")
)

// DoctorRepository defines data operations for Doctor profiles.
type DoctorRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Doctor, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Doctor, error)
	Update(ctx context.Context, doctor *model.Doctor) error
	Verify(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, specialty *string, onlyVerified bool, sortBy string, order string, offset int, limit int) ([]*model.Doctor, int, error)
}
