package model

import (
	"time"

	"github.com/google/uuid"
)

// Patient represents the domain entity for a patient profile.
type Patient struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	PhoneNumber *string    `json:"phone_number"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Gender      *string    `json:"gender"`
	BloodType   *string    `json:"blood_type"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}
