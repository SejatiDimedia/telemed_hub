package dto

// CreatePrescriptionRequest is the request body for issuing a prescription.
type CreatePrescriptionRequest struct {
	ConsultationID string                      `json:"consultation_id"`
	Items          []PrescriptionItemRequest   `json:"items"`
}

// PrescriptionItemRequest is a single line-item in the prescription request.
type PrescriptionItemRequest struct {
	MedicineID   string  `json:"medicine_id"`
	Dosage       string  `json:"dosage"`
	Quantity     int     `json:"quantity"`
	Instructions *string `json:"instructions,omitempty"`
}

// PrescriptionResponse is the full response for a prescription.
type PrescriptionResponse struct {
	ID             string                       `json:"id"`
	ConsultationID string                       `json:"consultation_id"`
	PatientID      string                       `json:"patient_id"`
	DoctorID       string                       `json:"doctor_id"`
	IssuedAt       string                       `json:"issued_at"`
	Status         string                       `json:"status"`
	Items          []PrescriptionItemResponse   `json:"items"`
	CreatedAt      string                       `json:"created_at"`
	UpdatedAt      string                       `json:"updated_at"`
}

// PrescriptionItemResponse is a single line-item in the prescription response.
type PrescriptionItemResponse struct {
	ID           string  `json:"id"`
	MedicineID   string  `json:"medicine_id"`
	MedicineName string  `json:"medicine_name"`
	Dosage       string  `json:"dosage"`
	Quantity     int     `json:"quantity"`
	Instructions *string `json:"instructions,omitempty"`
}
