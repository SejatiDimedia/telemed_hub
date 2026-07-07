package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	patientSvc "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/shared"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/repository"
)

var (
	ErrUnauthorized                  = errors.New("unauthorized access to medical record")
	ErrRecordNotFound                 = errors.New("medical record not found")
	ErrTreatmentRelationshipRequired = errors.New("treatment relationship is required to access or create medical records")
	ErrOnlyCreatorCanModify          = errors.New("only the issuing doctor or an admin can modify this medical record")
	ErrPatientIDRequired             = errors.New("patient_id filter is required for doctors and admins")
)

type MedicalRecordService interface {
	Create(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, req dto.CreateMedicalRecordRequest) (*dto.MedicalRecordResponse, error)
	GetByID(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, id uuid.UUID) (*dto.MedicalRecordResponse, error)
	List(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, filter dto.ListMedicalRecordsFilter) ([]*dto.MedicalRecordResponse, error)
	Update(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, id uuid.UUID, req dto.UpdateMedicalRecordRequest) (*dto.MedicalRecordResponse, error)
	Delete(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, id uuid.UUID) error
}

type MedicalRecordServiceImpl struct {
	repo       repository.MedicalRecordRepository
	patientSvc patientSvc.PatientService
	auditSvc   shared.AuditService
}

func NewMedicalRecordService(
	repo repository.MedicalRecordRepository,
	patientSvc patientSvc.PatientService,
	auditSvc shared.AuditService,
) *MedicalRecordServiceImpl {
	return &MedicalRecordServiceImpl{
		repo:       repo,
		patientSvc: patientSvc,
		auditSvc:   auditSvc,
	}
}

func (s *MedicalRecordServiceImpl) Create(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, req dto.CreateMedicalRecordRequest) (*dto.MedicalRecordResponse, error) {
	isAdmin := hasRole(roles, "admin")
	isDoctor := hasRole(roles, "doctor")

	if !isAdmin && !isDoctor {
		return nil, ErrUnauthorized
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		return nil, fmt.Errorf("invalid patient_id: %w", err)
	}

	// For doctors, verify treatment relationship exists
	if isDoctor && !isAdmin {
		hasRelationship, err := s.repo.HasTreatmentRelationship(ctx, actorID, patientID)
		if err != nil {
			return nil, err
		}
		if !hasRelationship {
			return nil, ErrTreatmentRelationshipRequired
		}
	}

	var consultationIDPtr *uuid.UUID
	if req.ConsultationID != nil && *req.ConsultationID != "" {
		cID, err := uuid.Parse(*req.ConsultationID)
		if err != nil {
			return nil, fmt.Errorf("invalid consultation_id: %w", err)
		}
		consultationIDPtr = &cID
	}

	var fileIDPtr *uuid.UUID
	if req.FileID != nil && *req.FileID != "" {
		fID, err := uuid.Parse(*req.FileID)
		if err != nil {
			return nil, fmt.Errorf("invalid file_id: %w", err)
		}
		fileIDPtr = &fID
	}

	recordID := uuid.New()
	rec := &model.MedicalRecord{
		ID:             recordID,
		PatientID:      patientID,
		ConsultationID: consultationIDPtr,
		RecordType:     req.RecordType,
		Content:        req.Content,
		FileID:         fileIDPtr,
		CreatedBy:      &actorID,
	}

	err = s.repo.Create(ctx, rec)
	if err != nil {
		return nil, err
	}

	// Write HIPAA Audit Log entry
	_ = s.auditSvc.Log(ctx, shared.AuditEntry{
		ActorID:    actorID,
		Action:     "medical_record.created",
		TargetType: "medical_records",
		TargetID:   recordID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata: map[string]any{
			"patient_id":  req.PatientID,
			"record_type": req.RecordType,
		},
	})

	return toResponse(rec), nil
}

func (s *MedicalRecordServiceImpl) GetByID(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, id uuid.UUID) (*dto.MedicalRecordResponse, error) {
	rec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrMedicalRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	isAdmin := hasRole(roles, "admin")
	isDoctor := hasRole(roles, "doctor")
	isPatient := hasRole(roles, "patient")

	if isPatient && !isAdmin && !isDoctor {
		patProfile, err := s.patientSvc.GetProfileByUserID(ctx, actorID)
		if err != nil {
			return nil, ErrUnauthorized
		}
		patientID, _ := uuid.Parse(patProfile.ID)
		if rec.PatientID != patientID {
			return nil, ErrUnauthorized
		}
	}

	if isDoctor && !isAdmin {
		hasRelationship, err := s.repo.HasTreatmentRelationship(ctx, actorID, rec.PatientID)
		if err != nil {
			return nil, err
		}
		if !hasRelationship {
			return nil, ErrTreatmentRelationshipRequired
		}
	}

	if !isAdmin && !isDoctor && !isPatient {
		return nil, ErrUnauthorized
	}

	// Write HIPAA Audit Log if read by doctor or admin
	if isDoctor || isAdmin {
		_ = s.auditSvc.Log(ctx, shared.AuditEntry{
			ActorID:    actorID,
			Action:     "medical_record.viewed",
			TargetType: "medical_records",
			TargetID:   id,
			IPAddress:  ipAddress,
			UserAgent:  userAgent,
			Metadata: map[string]any{
				"patient_id":  rec.PatientID.String(),
				"record_type": rec.RecordType,
			},
		})
	}

	return toResponse(rec), nil
}

func (s *MedicalRecordServiceImpl) List(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, filter dto.ListMedicalRecordsFilter) ([]*dto.MedicalRecordResponse, error) {
	isAdmin := hasRole(roles, "admin")
	isDoctor := hasRole(roles, "doctor")
	isPatient := hasRole(roles, "patient")

	if isPatient {
		patProfile, err := s.patientSvc.GetProfileByUserID(ctx, actorID)
		if err != nil {
			return nil, ErrUnauthorized
		}
		// Force filter to only match the patient's own ID
		pIDStr := patProfile.ID
		filter.PatientID = &pIDStr
	} else if isDoctor && !isAdmin {
		if filter.PatientID == nil || *filter.PatientID == "" {
			return nil, ErrPatientIDRequired
		}
		patientID, err := uuid.Parse(*filter.PatientID)
		if err != nil {
			return nil, fmt.Errorf("invalid patient_id format: %w", err)
		}
		hasRelationship, err := s.repo.HasTreatmentRelationship(ctx, actorID, patientID)
		if err != nil {
			return nil, err
		}
		if !hasRelationship {
			return nil, ErrTreatmentRelationshipRequired
		}
	} else if isAdmin {
		// Admin can view all or filter by patient
	} else {
		return nil, ErrUnauthorized
	}

	records, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Write HIPAA Audit Log if read by doctor or admin
	if (isDoctor || isAdmin) && filter.PatientID != nil {
		patID, _ := uuid.Parse(*filter.PatientID)
		_ = s.auditSvc.Log(ctx, shared.AuditEntry{
			ActorID:    actorID,
			Action:     "medical_record.viewed",
			TargetType: "patients",
			TargetID:   patID,
			IPAddress:  ipAddress,
			UserAgent:  userAgent,
			Metadata: map[string]any{
				"records_count": len(records),
			},
		})
	}

	respList := make([]*dto.MedicalRecordResponse, 0, len(records))
	for _, r := range records {
		respList = append(respList, toResponse(r))
	}
	return respList, nil
}

func (s *MedicalRecordServiceImpl) Update(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, id uuid.UUID, req dto.UpdateMedicalRecordRequest) (*dto.MedicalRecordResponse, error) {
	isAdmin := hasRole(roles, "admin")
	isDoctor := hasRole(roles, "doctor")

	if !isAdmin && !isDoctor {
		return nil, ErrUnauthorized
	}

	rec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrMedicalRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	if isDoctor && !isAdmin {
		if rec.CreatedBy == nil || *rec.CreatedBy != actorID {
			return nil, ErrOnlyCreatorCanModify
		}
	}

	var fileIDPtr *uuid.UUID
	if req.FileID != nil && *req.FileID != "" {
		fID, err := uuid.Parse(*req.FileID)
		if err != nil {
			return nil, fmt.Errorf("invalid file_id: %w", err)
		}
		fileIDPtr = &fID
	}

	rec.RecordType = req.RecordType
	rec.Content = req.Content
	rec.FileID = fileIDPtr
	rec.UpdatedBy = &actorID

	err = s.repo.Update(ctx, rec)
	if err != nil {
		return nil, err
	}

	// Write HIPAA Audit Log entry
	_ = s.auditSvc.Log(ctx, shared.AuditEntry{
		ActorID:    actorID,
		Action:     "medical_record.updated",
		TargetType: "medical_records",
		TargetID:   id,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata: map[string]any{
			"patient_id":  rec.PatientID.String(),
			"record_type": req.RecordType,
		},
	})

	return toResponse(rec), nil
}

func (s *MedicalRecordServiceImpl) Delete(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, id uuid.UUID) error {
	isAdmin := hasRole(roles, "admin")
	if !isAdmin {
		return ErrUnauthorized
	}

	rec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrMedicalRecordNotFound) {
			return ErrRecordNotFound
		}
		return err
	}

	err = s.repo.SoftDelete(ctx, id, actorID)
	if err != nil {
		return err
	}

	// Write HIPAA Audit Log entry
	_ = s.auditSvc.Log(ctx, shared.AuditEntry{
		ActorID:    actorID,
		Action:     "medical_record.deleted",
		TargetType: "medical_records",
		TargetID:   id,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata: map[string]any{
			"patient_id":  rec.PatientID.String(),
			"record_type": rec.RecordType,
		},
	})

	return nil
}

func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

func toResponse(r *model.MedicalRecord) *dto.MedicalRecordResponse {
	var cIDStr *string
	if r.ConsultationID != nil {
		s := r.ConsultationID.String()
		cIDStr = &s
	}
	var fIDStr *string
	if r.FileID != nil {
		s := r.FileID.String()
		fIDStr = &s
	}

	return &dto.MedicalRecordResponse{
		ID:             r.ID.String(),
		PatientID:      r.PatientID.String(),
		ConsultationID: cIDStr,
		RecordType:     r.RecordType,
		Content:        r.Content,
		FileID:         fIDStr,
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      r.UpdatedAt.Format(time.RFC3339),
	}
}
