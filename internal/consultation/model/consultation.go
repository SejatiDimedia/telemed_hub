package model

import (
	"time"

	"github.com/google/uuid"
)

// Consultation represents the clinical session associated with an appointment.
type Consultation struct {
	ID            uuid.UUID
	AppointmentID uuid.UUID
	Status        string
	Notes         *string
	StartedAt     *time.Time
	EndedAt       *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
