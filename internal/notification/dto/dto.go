package dto

import "time"

type NotificationResponse struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	Channel         string         `json:"channel"`
	Type            string         `json:"type"`
	Status          string         `json:"status"`
	Payload         map[string]any `json:"payload"`
	RetryCount      int            `json:"retry_count"`
	LastAttemptedAt *time.Time     `json:"last_attempted_at,omitempty"`
	SentAt          *time.Time     `json:"sent_at,omitempty"`
	FailedAt        *time.Time     `json:"failed_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}
