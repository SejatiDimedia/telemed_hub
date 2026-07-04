package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
)

// DoctorService defines doctor use cases.
type DoctorService interface {
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*dto.DoctorResponse, error)
	GetProfileByID(ctx context.Context, id uuid.UUID) (*dto.DoctorResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateDoctorRequest) (*dto.DoctorResponse, error)
	VerifyDoctor(ctx context.Context, adminUserID uuid.UUID, doctorID uuid.UUID, ipAddress string, userAgent string) error
	ListDoctors(ctx context.Context, specialty *string, onlyVerified bool, sortBy string, order string, page int, limit int) ([]*dto.DoctorResponse, int, error)
}
