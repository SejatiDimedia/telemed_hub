package model

import (
	"time"

	"github.com/google/uuid"
)

// Doctor represents the domain entity for a doctor profile.
type Doctor struct {
	ID                   uuid.UUID  `json:"id"`
	UserID               uuid.UUID  `json:"user_id"`
	Email                string     `json:"email"`
	FullName             string     `json:"full_name"`
	PhoneNumber          *string    `json:"phone_number"`
	Specialty            *string    `json:"specialty"`
	LicenseNumber        *string    `json:"license_number"`
	IsCredentialVerified bool       `json:"is_credential_verified"`
	ConsultationFee      int64      `json:"consultation_fee"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `json:"deleted_at"`
}
