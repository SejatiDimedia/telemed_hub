package wallet

import (
	"context"

	"github.com/google/uuid"
)

// WalletService defines the interface for interacting with the wallet module.
type WalletService interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (int64, error)
	Deduct(ctx context.Context, userID uuid.UUID, amount int64, description string) error
	Refund(ctx context.Context, userID uuid.UUID, amount int64, description string) error
}
