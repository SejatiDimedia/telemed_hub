package dto

type UpdateNotesRequest struct {
	Notes string `json:"notes"`
}

type ConsultationResponse struct {
	ID            string  `json:"id"`
	AppointmentID string  `json:"appointment_id"`
	Status        string  `json:"status"`
	Notes         *string `json:"notes,omitempty"`
	StartedAt     *string `json:"started_at,omitempty"`
	EndedAt       *string `json:"ended_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}
