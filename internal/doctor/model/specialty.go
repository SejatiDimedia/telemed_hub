package model

import (
	"time"

	"github.com/google/uuid"
)

// Specialty represents a medical specialty category (e.g. Cardiologist).
type Specialty struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	ImageIcon   string     `json:"image_icon"`
	Description *string    `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}
