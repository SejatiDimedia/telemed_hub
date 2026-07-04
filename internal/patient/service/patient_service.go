package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/mapper"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/validator"
)

type PatientServiceImpl struct {
	repo repository.PatientRepository
}

func NewPatientService(repo repository.PatientRepository) *PatientServiceImpl {
	return &PatientServiceImpl{repo: repo}
}

func (s *PatientServiceImpl) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*dto.PatientResponse, error) {
	patient, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapper.ToResponse(patient), nil
}

func (s *PatientServiceImpl) GetProfileByID(ctx context.Context, id uuid.UUID) (*dto.PatientResponse, error) {
	patient, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapper.ToResponse(patient), nil
}

func (s *PatientServiceImpl) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdatePatientRequest) (*dto.PatientResponse, error) {
	// Validate updates DTO
	if err := validator.ValidateUpdatePatient(req); err != nil {
		return nil, err
	}

	// Fetch existing patient profile
	patient, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Parsing properties
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	gender := strings.ToLower(strings.TrimSpace(req.Gender))
	phone := strings.TrimSpace(req.PhoneNumber)

	var bloodType *string
	if req.BloodType != nil {
		bt := strings.ToUpper(strings.TrimSpace(*req.BloodType))
		if bt != "" {
			bloodType = &bt
		}
	}

	// Assign updates
	patient.DateOfBirth = &dob
	patient.Gender = &gender
	patient.BloodType = bloodType
	patient.PhoneNumber = &phone

	// Save to DB
	if err := s.repo.Update(ctx, patient); err != nil {
		return nil, err
	}

	return mapper.ToResponse(patient), nil
}
