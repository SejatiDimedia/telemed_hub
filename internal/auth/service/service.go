package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/dto"
)

type AuthService interface {
	// Register registers a new user (patient or doctor) and initializes their profile.
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error)

	// Login authenticates a user and returns access and refresh tokens.
	Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.AuthResponse, error)

	// RefreshToken rotates refresh tokens and returns a new access token.
	RefreshToken(ctx context.Context, req dto.RefreshRequest, ipAddress, userAgent string) (*dto.AuthResponse, error)

	// Logout revokes the given refresh token (single device or all devices).
	Logout(ctx context.Context, userID uuid.UUID, rawRefreshToken string, allDevices bool) error

	// GetUserByID returns user info by ID.
	GetUserByID(ctx context.Context, id uuid.UUID) (*dto.UserResponse, error)
}
