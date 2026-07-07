package model

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Channel         string // 'email', 'push', 'sms'
	Type            string // 'appointment_reminder', 'order_status', 'appointment_confirmed', etc.
	Status          string // 'pending', 'sent', 'failed'
	Payload         map[string]any
	RetryCount      int
	LastAttemptedAt *time.Time
	SentAt          *time.Time
	FailedAt        *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
