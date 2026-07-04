package dto

// RegisterRequest holds registration inputs.
type RegisterRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"` // "patient" or "doctor"
}

// LoginRequest holds credentials for sign-in.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest holds the refresh token for session renewals.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
