package dto

// UpdateDoctorRequest defines the payload for updating doctor profiles.
type UpdateDoctorRequest struct {
	Specialty       string `json:"specialty"`
	LicenseNumber   string `json:"license_number"`
	ConsultationFee int64  `json:"consultation_fee"`
	PhoneNumber     string `json:"phone_number"`
}
