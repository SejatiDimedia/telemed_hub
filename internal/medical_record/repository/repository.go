package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/model"
)

var (
	ErrMedicalRecordNotFound = errors.New("medical record not found")
)

type MedicalRecordRepository interface {
	Create(ctx context.Context, rec *model.MedicalRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.MedicalRecord, error)
	List(ctx context.Context, filter dto.ListMedicalRecordsFilter) ([]*model.MedicalRecord, error)
	Update(ctx context.Context, rec *model.MedicalRecord) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	HasTreatmentRelationship(ctx context.Context, doctorUserID, patientID uuid.UUID) (bool, error)
}
