package dto

type AppointmentResponse struct {
	ID             string  `json:"id"`
	PatientID      string  `json:"patient_id"`
	DoctorID       string  `json:"doctor_id"`
	AvailabilityID string  `json:"availability_id"`
	Status         string  `json:"status"`
	ScheduledAt    string  `json:"scheduled_at"`
	CancelledAt    *string `json:"cancelled_at,omitempty"`
	CancelReason   *string `json:"cancel_reason,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}
