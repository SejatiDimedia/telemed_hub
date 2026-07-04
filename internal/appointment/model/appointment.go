package model

import (
	"time"

	"github.com/google/uuid"
)

// Appointment represents the core booking record between a patient and a doctor.
type Appointment struct {
	ID             uuid.UUID
	PatientID      uuid.UUID
	DoctorID       uuid.UUID
	AvailabilityID uuid.UUID
	Status         string
	ScheduledAt    time.Time
	CancelledAt    *time.Time
	CancelReason   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
