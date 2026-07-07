package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/model"
)

var (
	ErrWalletNotFound = errors.New("wallet not found")
)

type WalletRepository interface {
	GetOrCreateWallet(ctx context.Context, tx pgx.Tx, patientID uuid.UUID) (*model.Wallet, error)
	GetWalletByUserID(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Wallet, error)
	GetWalletByUserIDForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Wallet, error)
	UpdateWalletBalance(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, balance float64) error
	CreateTransaction(ctx context.Context, tx pgx.Tx, txRecord *model.WalletTransaction) error
	GetTransactionByIdempotencyKey(ctx context.Context, tx pgx.Tx, key string) (*model.WalletTransaction, error)
	ListTransactions(ctx context.Context, walletID uuid.UUID, typeFilter *string, page, limit int) ([]*model.WalletTransaction, int, error)
	VerifyLedgerConsistency(ctx context.Context, walletID uuid.UUID) (float64, error)
}
