package dto

type UpdateNotesRequest struct {
	Notes string `json:"notes"`
}

type ConsultationResponse struct {
	ID             string  `json:"id"`
	AppointmentID  string  `json:"appointment_id"`
	// DoctorID and PatientID are resolved from the linked appointment.
	// Populated by ConsultationService.GetByID — allows downstream modules
	// (e.g. prescription) to resolve ownership without importing appointment.
	DoctorID       string  `json:"doctor_id"`
	PatientID      string  `json:"patient_id"`
	Status         string  `json:"status"`
	Notes          *string `json:"notes,omitempty"`
	StartedAt      *string `json:"started_at,omitempty"`
	EndedAt        *string `json:"ended_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}
