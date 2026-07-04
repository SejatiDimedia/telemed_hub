package dto

// DoctorResponse defines the API response payload for doctor profiles.
type DoctorResponse struct {
	ID                   string  `json:"id"`
	UserID               string  `json:"user_id"`
	Email                string  `json:"email"`
	FullName             string  `json:"full_name"`
	PhoneNumber          *string `json:"phone_number,omitempty"`
	Specialty            *string `json:"specialty"`
	LicenseNumber        *string `json:"license_number,omitempty"`
	IsCredentialVerified bool    `json:"is_credential_verified"`
	ConsultationFee      int64   `json:"consultation_fee"`
}

// SanitizeForPublic returns a copy of DoctorResponse with sensitive credentials omitted.
func (d DoctorResponse) SanitizeForPublic() DoctorResponse {
	d.PhoneNumber = nil
	d.LicenseNumber = nil
	return d
}
