package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/model"
)

type AuthRepository interface {
	// CreateUserWithRoleAndProfile creates user record, maps role, and initializes doctor/patient profile in a transaction.
	CreateUserWithRoleAndProfile(ctx context.Context, user *model.User, roleName string) error

	// GetUserByEmail retrieves a user by their email address.
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)

	// GetUserByID retrieves a user by their UUID.
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)

	// GetUserRoles retrieves names of all roles assigned to the user.
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)

	// SaveRefreshToken persists a hashed refresh token.
	SaveRefreshToken(ctx context.Context, token *model.RefreshToken) error

	// GetRefreshTokenByHash retrieves token details matching the hash.
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)

	// RevokeRefreshToken marks a single refresh token as revoked.
	RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error

	// RevokeAllUserRefreshTokens marks all active refresh tokens of a user as revoked (all devices logout).
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error

	// GetActiveRefreshTokens returns all non-revoked, non-expired refresh tokens of a user.
	GetActiveRefreshTokens(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error)
}
