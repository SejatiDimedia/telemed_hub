package model

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID
	ActorID    uuid.UUID
	Action     string
	TargetType string
	TargetID   uuid.UUID
	IPAddress  string
	UserAgent  *string
	Metadata   map[string]any
	CreatedAt  time.Time
}
