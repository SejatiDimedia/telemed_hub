package validator

import (
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/dto"
)

var (
	ErrInvalidPatientID  = errors.New("patient_id is required and must be a valid UUID")
	ErrInvalidRecordType = errors.New("record_type is required and must be one of: 'diagnosis', 'allergy', 'lab_result', 'note'")
	ErrContentRequired   = errors.New("content is required and cannot be empty")
)

func ValidateCreate(req dto.CreateMedicalRecordRequest) error {
	if _, err := uuid.Parse(req.PatientID); err != nil {
		return ErrInvalidPatientID
	}
	if !isValidRecordType(req.RecordType) {
		return ErrInvalidRecordType
	}
	if req.Content == "" {
		return ErrContentRequired
	}
	if req.ConsultationID != nil && *req.ConsultationID != "" {
		if _, err := uuid.Parse(*req.ConsultationID); err != nil {
			return errors.New("invalid consultation_id format")
		}
	}
	if req.FileID != nil && *req.FileID != "" {
		if _, err := uuid.Parse(*req.FileID); err != nil {
			return errors.New("invalid file_id format")
		}
	}
	return nil
}

func ValidateUpdate(req dto.UpdateMedicalRecordRequest) error {
	if !isValidRecordType(req.RecordType) {
		return ErrInvalidRecordType
	}
	if req.Content == "" {
		return ErrContentRequired
	}
	if req.FileID != nil && *req.FileID != "" {
		if _, err := uuid.Parse(*req.FileID); err != nil {
			return errors.New("invalid file_id format")
		}
	}
	return nil
}

func isValidRecordType(t string) bool {
	switch t {
	case "diagnosis", "allergy", "lab_result", "note":
		return true
	}
	return false
}
