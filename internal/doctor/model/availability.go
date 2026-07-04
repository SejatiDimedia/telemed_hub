package model

import (
	"time"

	"github.com/google/uuid"
)

// Availability represents a time slot for doctor availability.
type Availability struct {
	ID        uuid.UUID
	DoctorID  uuid.UUID
	StartTime time.Time
	EndTime   time.Time
	IsBooked  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
