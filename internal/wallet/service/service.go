package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/dto"
)

// WalletService defines the interface for interacting with the wallet module.
type WalletService interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (int64, error)
	Deduct(ctx context.Context, userID uuid.UUID, amount int64, description string) error
	Refund(ctx context.Context, userID uuid.UUID, amount int64, description string) error

	// REST API operations for the wallet module
	GetBalanceDetails(ctx context.Context, userID uuid.UUID) (*dto.WalletResponse, error)
	TopUp(ctx context.Context, userID uuid.UUID, amount float64, idempotencyKey *string) (*dto.TransactionResponse, error)
	ListTransactions(ctx context.Context, userID uuid.UUID, typeFilter *string, page, limit int) ([]*dto.TransactionResponse, int, error)
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (*dto.TransactionResponse, error)

	// Transactional methods for atomic multi-module operations
	DeductTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, amount int64, description string, idempotencyKey *string) error
	RefundTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, amount int64, description string) error
}
