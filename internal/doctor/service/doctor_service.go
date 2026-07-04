package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/mapper"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
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

func (s *DoctorServiceImpl) AddAvailability(ctx context.Context, doctorUserID uuid.UUID, req dto.CreateAvailabilityRequest) (*dto.AvailabilityResponse, error) {
	// 1. Get Doctor profile
	doctor, err := s.repo.GetByUserID(ctx, doctorUserID)
	if err != nil {
		return nil, err
	}

	// 2. Validate input and parse times
	startTime, endTime, err := validator.ValidateCreateAvailability(req)
	if err != nil {
		return nil, err
	}

	// 3. Check for overlapping slot
	overlapping, err := s.repo.CheckOverlappingSlot(ctx, doctor.ID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	if overlapping {
		return nil, ErrOverlappingSlot
	}

	// 4. Save slot
	slot := &model.Availability{
		DoctorID:  doctor.ID,
		StartTime: startTime,
		EndTime:   endTime,
		IsBooked:  false,
	}

	err = s.repo.CreateAvailability(ctx, slot)
	if err != nil {
		return nil, err
	}

	return mapper.ToAvailabilityResponse(slot), nil
}

func (s *DoctorServiceImpl) RemoveAvailability(ctx context.Context, doctorUserID uuid.UUID, slotID uuid.UUID) error {
	// 1. Get Doctor profile
	doctor, err := s.repo.GetByUserID(ctx, doctorUserID)
	if err != nil {
		return err
	}

	// 2. Perform deletion (repo checks ownership and booking status)
	return s.repo.DeleteAvailability(ctx, doctor.ID, slotID)
}

func (s *DoctorServiceImpl) GetAvailability(ctx context.Context, doctorID uuid.UUID, startTimeStr, endTimeStr string, isBooked *bool) ([]*dto.AvailabilityResponse, error) {
	var startTime, endTime time.Time
	var err error

	if startTimeStr == "" {
		startTime = time.Now().UTC()
	} else {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return nil, err
		}
	}

	if endTimeStr == "" {
		endTime = startTime.Add(7 * 24 * time.Hour)
	} else {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, err
		}
	}

	slots, err := s.repo.ListAvailability(ctx, doctorID, startTime, endTime, isBooked)
	if err != nil {
		return nil, err
	}

	return mapper.ToAvailabilityResponseList(slots), nil
}
