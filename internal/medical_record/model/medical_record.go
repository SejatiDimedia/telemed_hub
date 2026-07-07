package model

import (
	"time"

	"github.com/google/uuid"
)

type MedicalRecord struct {
	ID             uuid.UUID
	PatientID      uuid.UUID
	ConsultationID *uuid.UUID
	RecordType     string // 'diagnosis', 'allergy', 'lab_result', 'note'
	Content        string
	FileID         *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
	CreatedBy      *uuid.UUID
	UpdatedBy      *uuid.UUID
	DeletedBy      *uuid.UUID
}
