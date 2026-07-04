package dto

// RegisterResponse is returned after successful registration.
type RegisterResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// AuthResponse holds JWT tokens and session expiration.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// UserResponse represents user profile/session details returned by GET /auth/me.
type UserResponse struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	FullName string   `json:"full_name"`
}
