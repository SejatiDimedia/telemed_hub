package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/mapper"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/validator"
	"github.com/timurdianradhasejati/telemed_hub/internal/shared"
)

type DoctorServiceImpl struct {
	repo         repository.DoctorRepository
	auditService shared.AuditService
}

func NewDoctorService(repo repository.DoctorRepository, auditService shared.AuditService) *DoctorServiceImpl {
	return &DoctorServiceImpl{
		repo:         repo,
		auditService: auditService,
	}
}

func (s *DoctorServiceImpl) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*dto.DoctorResponse, error) {
	doctor, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapper.ToResponse(doctor), nil
}

func (s *DoctorServiceImpl) GetProfileByID(ctx context.Context, id uuid.UUID) (*dto.DoctorResponse, error) {
	doctor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapper.ToResponse(doctor), nil
}

func (s *DoctorServiceImpl) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateDoctorRequest) (*dto.DoctorResponse, error) {
	if err := validator.ValidateUpdateDoctor(req); err != nil {
		return nil, err
	}

	doctor, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	specialty := strings.TrimSpace(req.Specialty)
	license := strings.TrimSpace(req.LicenseNumber)
	phone := strings.TrimSpace(req.PhoneNumber)

	doctor.Specialty = &specialty
	doctor.LicenseNumber = &license
	doctor.ConsultationFee = req.ConsultationFee
	doctor.PhoneNumber = &phone

	if err := s.repo.Update(ctx, doctor); err != nil {
		return nil, err
	}

	return mapper.ToResponse(doctor), nil
}

func (s *DoctorServiceImpl) VerifyDoctor(ctx context.Context, adminUserID uuid.UUID, doctorID uuid.UUID, ipAddress string, userAgent string) error {
	// Verify in DB
	if err := s.repo.Verify(ctx, doctorID); err != nil {
		return err
	}

	// Write audit log entry via shared AuditService
	if s.auditService != nil {
		_ = s.auditService.Log(ctx, shared.AuditEntry{
			ActorID:    adminUserID,
			Action:     "doctor.verified",
			TargetType: "doctors",
			TargetID:   doctorID,
			IPAddress:  ipAddress,
			UserAgent:  userAgent,
			Metadata: map[string]any{
				"verified_at":            time.Now().UTC().Format(time.RFC3339),
				"is_credential_verified": true,
			},
		})
	}

	return nil
}

func (s *DoctorServiceImpl) ListDoctors(ctx context.Context, specialty *string, onlyVerified bool, sortBy string, order string, page int, limit int) ([]*dto.DoctorResponse, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}

	offset := (page - 1) * limit

	doctors, totalItems, err := s.repo.List(ctx, specialty, onlyVerified, sortBy, order, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	return mapper.ToResponseList(doctors), totalItems, nil
}
