package dto

type CreateAppointmentRequest struct {
	DoctorID       string `json:"doctor_id"`
	AvailabilityID string `json:"availability_id"`
}

type RescheduleAppointmentRequest struct {
	NewAvailabilityID string `json:"new_availability_id"`
}

type CancelAppointmentRequest struct {
	CancelReason string `json:"cancel_reason"`
}
