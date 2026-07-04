package model

import (
	"time"

	"github.com/google/uuid"
)

// Prescription is the header record of a doctor-issued prescription.
type Prescription struct {
	ID             uuid.UUID
	ConsultationID uuid.UUID
	PatientID      uuid.UUID
	DoctorID       uuid.UUID
	IssuedAt       time.Time
	Status         string // active | fulfilled | expired
	Items          []PrescriptionItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      *uuid.UUID
	UpdatedBy      *uuid.UUID
	DeletedAt      *time.Time
	DeletedBy      *uuid.UUID
}

// PrescriptionItem is a single line-item of medicines in a prescription.
type PrescriptionItem struct {
	ID             uuid.UUID
	PrescriptionID uuid.UUID
	MedicineID     uuid.UUID
	MedicineName   string // denormalized from medicines table for response
	Dosage         string
	Quantity       int
	Instructions   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
