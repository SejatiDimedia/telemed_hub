package dto

// UpdatePatientRequest defines the payload for updating patient profiles.
type UpdatePatientRequest struct {
	DateOfBirth string  `json:"date_of_birth"`
	Gender      string  `json:"gender"`
	BloodType   *string `json:"blood_type"`
	PhoneNumber string  `json:"phone_number"`
}
