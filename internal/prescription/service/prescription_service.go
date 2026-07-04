package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	consultationSvc "github.com/timurdianradhasejati/telemed_hub/internal/consultation/service"
	doctorSvc "github.com/timurdianradhasejati/telemed_hub/internal/doctor/service"
	patientSvc "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/repository"
)

// PrescriptionServiceImpl is the concrete prescription use-case implementation.
type PrescriptionServiceImpl struct {
	repo            repository.PrescriptionRepository
	consultationSvc consultationSvc.ConsultationService
	doctorSvc       doctorSvc.DoctorService
	patientSvc      patientSvc.PatientService
}

// NewPrescriptionService wires the service with its dependencies.
func NewPrescriptionService(
	repo repository.PrescriptionRepository,
	consultationSvc consultationSvc.ConsultationService,
	doctorSvc doctorSvc.DoctorService,
	patientSvc patientSvc.PatientService,
) *PrescriptionServiceImpl {
	return &PrescriptionServiceImpl{
		repo:            repo,
		consultationSvc: consultationSvc,
		doctorSvc:       doctorSvc,
		patientSvc:      patientSvc,
	}
}

// Issue creates a new prescription for a consultation.
// Business rules:
//  1. Consultation must be in_progress or completed (not scheduled/cancelled).
//  2. The calling doctor must be the assigned doctor on the consultation.
//  3. Items list must not be empty (validated at handler/validator layer).
func (s *PrescriptionServiceImpl) Issue(ctx context.Context, doctorUserID uuid.UUID, req dto.CreatePrescriptionRequest) (*dto.PrescriptionResponse, error) {
	consultationID, _ := uuid.Parse(req.ConsultationID) // already validated by caller

	// 1. Fetch doctor profile to resolve DoctorID (doctors table PK)
	docProfile, err := s.doctorSvc.GetProfileByUserID(ctx, doctorUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve doctor profile: %w", err)
	}
	doctorID, _ := uuid.Parse(docProfile.ID)

	// 2. Fetch consultation — this also verifies the caller is authorized (doctor role) and
	//    returns DoctorID + PatientID populated from the linked appointment.
	consultation, err := s.consultationSvc.GetByID(ctx, consultationID, doctorUserID, []string{"doctor"})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch consultation: %w", err)
	}

	// 3. Validate consultation status: must be in_progress or completed
	if consultation.Status != "in_progress" && consultation.Status != "completed" {
		return nil, ErrInvalidConsultationStatus
	}

	// 4. Verify the calling doctor is THE assigned doctor on this consultation
	apptDoctorID, _ := uuid.Parse(consultation.DoctorID)
	if doctorID != apptDoctorID {
		return nil, ErrUnauthorized
	}

	patientID, _ := uuid.Parse(consultation.PatientID)

	// 5. Build domain model
	pres := &model.Prescription{
		ConsultationID: consultationID,
		PatientID:      patientID,
		DoctorID:       doctorID,
		Status:         "active",
		CreatedBy:      &doctorUserID,
	}
	for _, it := range req.Items {
		medicineID, _ := uuid.Parse(it.MedicineID)
		pres.Items = append(pres.Items, model.PrescriptionItem{
			MedicineID:   medicineID,
			Dosage:       it.Dosage,
			Quantity:     it.Quantity,
			Instructions: it.Instructions,
		})
	}

	// 6. Persist header + items atomically
	if err := s.repo.Create(ctx, pres); err != nil {
		return nil, fmt.Errorf("failed to save prescription: %w", err)
	}

	// 7. Reload to return with denormalized medicine names from the JOIN
	saved, err := s.repo.GetByID(ctx, pres.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload prescription after create: %w", err)
	}

	return toResponse(saved), nil
}

// GetByID returns a prescription, enforcing role-based access:
//   - patient: only their own prescriptions
//   - doctor: only prescriptions they issued
//   - pharmacy_staff / admin: any prescription
func (s *PrescriptionServiceImpl) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.PrescriptionResponse, error) {
	pres, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if hasRole(roles, "admin") || hasRole(roles, "pharmacy_staff") {
		return toResponse(pres), nil
	}

	if hasRole(roles, "doctor") {
		docProfile, err := s.doctorSvc.GetProfileByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve doctor profile: %w", err)
		}
		doctorID, _ := uuid.Parse(docProfile.ID)
		if pres.DoctorID != doctorID {
			return nil, ErrUnauthorized
		}
		return toResponse(pres), nil
	}

	if hasRole(roles, "patient") {
		patProfile, err := s.patientSvc.GetProfileByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve patient profile: %w", err)
		}
		patientID, _ := uuid.Parse(patProfile.ID)
		if pres.PatientID != patientID {
			return nil, ErrUnauthorized
		}
		return toResponse(pres), nil
	}

	return nil, ErrUnauthorized
}

// List returns prescriptions scoped to the calling user's role.
func (s *PrescriptionServiceImpl) List(ctx context.Context, userID uuid.UUID, roles []string) ([]*dto.PrescriptionResponse, error) {
	if hasRole(roles, "admin") || hasRole(roles, "pharmacy_staff") {
		// Pharmacy staff / admin: return empty list — they should look up by prescription ID.
		// A global listing endpoint without patient/doctor scoping is out of Sprint 6 scope.
		return []*dto.PrescriptionResponse{}, nil
	}

	if hasRole(roles, "doctor") {
		docProfile, err := s.doctorSvc.GetProfileByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve doctor profile: %w", err)
		}
		doctorID, _ := uuid.Parse(docProfile.ID)
		records, err := s.repo.ListByDoctorID(ctx, doctorID)
		if err != nil {
			return nil, err
		}
		return toResponseList(records), nil
	}

	if hasRole(roles, "patient") {
		patProfile, err := s.patientSvc.GetProfileByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve patient profile: %w", err)
		}
		patientID, _ := uuid.Parse(patProfile.ID)
		records, err := s.repo.ListByPatientID(ctx, patientID)
		if err != nil {
			return nil, err
		}
		return toResponseList(records), nil
	}

	return nil, ErrUnauthorized
}

// hasRole checks if a given role is present in the list.
func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

// toResponse maps a domain Prescription to a PrescriptionResponse DTO.
func toResponse(pres *model.Prescription) *dto.PrescriptionResponse {
	resp := &dto.PrescriptionResponse{
		ID:             pres.ID.String(),
		ConsultationID: pres.ConsultationID.String(),
		PatientID:      pres.PatientID.String(),
		DoctorID:       pres.DoctorID.String(),
		IssuedAt:       pres.IssuedAt.Format(time.RFC3339),
		Status:         pres.Status,
		CreatedAt:      pres.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      pres.UpdatedAt.Format(time.RFC3339),
		Items:          []dto.PrescriptionItemResponse{},
	}
	for _, item := range pres.Items {
		resp.Items = append(resp.Items, dto.PrescriptionItemResponse{
			ID:           item.ID.String(),
			MedicineID:   item.MedicineID.String(),
			MedicineName: item.MedicineName,
			Dosage:       item.Dosage,
			Quantity:     item.Quantity,
			Instructions: item.Instructions,
		})
	}
	return resp
}

// toResponseList maps a slice of Prescriptions to response DTOs (header only, no items).
func toResponseList(records []*model.Prescription) []*dto.PrescriptionResponse {
	result := make([]*dto.PrescriptionResponse, 0, len(records))
	for _, p := range records {
		result = append(result, toResponse(p))
	}
	return result
}
