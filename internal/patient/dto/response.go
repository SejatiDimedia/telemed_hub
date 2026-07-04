package dto

// PatientResponse defines the API response payload for patient profiles.
type PatientResponse struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Email       string  `json:"email"`
	FullName    string  `json:"full_name"`
	PhoneNumber *string `json:"phone_number"`
	DateOfBirth *string `json:"date_of_birth"`
	Gender      *string `json:"gender"`
	BloodType   *string `json:"blood_type"`
}
