package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/repository"
)

var (
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrMaxTopUpExceeded    = errors.New("top-up amount exceeds maximum allowed limit")
)

type WalletServiceImpl struct {
	repo     repository.WalletRepository
	db       *pgxpool.Pool
	maxTopUp float64
}

func NewWalletService(repo repository.WalletRepository, db *pgxpool.Pool) *WalletServiceImpl {
	maxVal := 10000000.00 // Default 10 million IDR
	if maxStr := os.Getenv("WALLET_MAX_TOPUP_AMOUNT"); maxStr != "" {
		if parsed, err := strconv.ParseFloat(maxStr, 64); err == nil && parsed > 0 {
			maxVal = parsed
		}
	}

	return &WalletServiceImpl{
		repo:     repo,
		db:       db,
		maxTopUp: maxVal,
	}
}

// GetBalance returns the balance as an int64 (compatible with the standard interface).
func (s *WalletServiceImpl) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	w, err := s.repo.GetWalletByUserID(ctx, nil, userID)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return int64(w.Balance), nil
}

// Deduct processes internal module balance deductions.
func (s *WalletServiceImpl) Deduct(ctx context.Context, userID uuid.UUID, amount int64, description string) error {
	if amount <= 0 {
		return errors.New("deduction amount must be greater than 0")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	w, err := s.repo.GetWalletByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		return err
	}

	if w.Balance < float64(amount) {
		return ErrInsufficientBalance
	}

	newBalance := w.Balance - float64(amount)
	err = s.repo.UpdateWalletBalance(ctx, tx, w.ID, newBalance)
	if err != nil {
		return err
	}

	txType := "consultation_payment"
	if strings.Contains(strings.ToLower(description), "order") {
		txType = "order_payment"
	}

	txRecord := &model.WalletTransaction{
		WalletID:     w.ID,
		Type:         txType,
		Amount:       float64(amount),
		BalanceAfter: newBalance,
	}

	err = s.repo.CreateTransaction(ctx, tx, txRecord)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Refund processes internal module balance refunds.
func (s *WalletServiceImpl) Refund(ctx context.Context, userID uuid.UUID, amount int64, description string) error {
	if amount <= 0 {
		return errors.New("refund amount must be greater than 0")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	w, err := s.repo.GetWalletByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		return err
	}

	newBalance := w.Balance + float64(amount)
	err = s.repo.UpdateWalletBalance(ctx, tx, w.ID, newBalance)
	if err != nil {
		return err
	}

	txRecord := &model.WalletTransaction{
		WalletID:     w.ID,
		Type:         "refund",
		Amount:       float64(amount),
		BalanceAfter: newBalance,
	}

	err = s.repo.CreateTransaction(ctx, tx, txRecord)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetBalanceDetails returns WalletResponse containing balance and currency.
func (s *WalletServiceImpl) GetBalanceDetails(ctx context.Context, userID uuid.UUID) (*dto.WalletResponse, error) {
	w, err := s.repo.GetWalletByUserID(ctx, nil, userID)
	if err != nil {
		return nil, err
	}

	return &dto.WalletResponse{
		Balance:  w.Balance,
		Currency: "IDR",
	}, nil
}

// TopUp processes a wallet top-up request. Conforms to idempotency check rules.
func (s *WalletServiceImpl) TopUp(ctx context.Context, userID uuid.UUID, amount float64, idempotencyKey *string) (*dto.TransactionResponse, error) {
	if amount <= 0 {
		return nil, errors.New("top-up amount must be greater than 0")
	}
	if amount > s.maxTopUp {
		return nil, ErrMaxTopUpExceeded
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Check Idempotency Key
	if idempotencyKey != nil && *idempotencyKey != "" {
		existing, err := s.repo.GetTransactionByIdempotencyKey(ctx, tx, *idempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return toTransactionResponse(existing), nil
		}
	}

	// Lock the wallet
	w, err := s.repo.GetWalletByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	newBalance := w.Balance + amount
	err = s.repo.UpdateWalletBalance(ctx, tx, w.ID, newBalance)
	if err != nil {
		return nil, err
	}

	txRecord := &model.WalletTransaction{
		WalletID:       w.ID,
		Type:           "top_up",
		Amount:         amount,
		BalanceAfter:   newBalance,
		IdempotencyKey: idempotencyKey,
	}

	err = s.repo.CreateTransaction(ctx, tx, txRecord)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return toTransactionResponse(txRecord), nil
}

func (s *WalletServiceImpl) ListTransactions(ctx context.Context, userID uuid.UUID, typeFilter *string, page, limit int) ([]*dto.TransactionResponse, int, error) {
	w, err := s.repo.GetWalletByUserID(ctx, nil, userID)
	if err != nil {
		return nil, 0, err
	}

	txs, total, err := s.repo.ListTransactions(ctx, w.ID, typeFilter, page, limit)
	if err != nil {
		return nil, 0, err
	}

	respList := make([]*dto.TransactionResponse, 0, len(txs))
	for _, t := range txs {
		respList = append(respList, toTransactionResponse(t))
	}
	return respList, total, nil
}

func (s *WalletServiceImpl) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*dto.TransactionResponse, error) {
	t, err := s.repo.GetTransactionByIdempotencyKey(ctx, nil, key)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}
	return toTransactionResponse(t), nil
}

func (s *WalletServiceImpl) DeductTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, amount int64, description string, idempotencyKey *string) error {
	if amount <= 0 {
		return errors.New("deduction amount must be greater than 0")
	}

	w, err := s.repo.GetWalletByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		return err
	}

	if w.Balance < float64(amount) {
		return ErrInsufficientBalance
	}

	newBalance := w.Balance - float64(amount)
	err = s.repo.UpdateWalletBalance(ctx, tx, w.ID, newBalance)
	if err != nil {
		return err
	}

	txType := "consultation_payment"
	if strings.Contains(strings.ToLower(description), "order") {
		txType = "order_payment"
	}

	txRecord := &model.WalletTransaction{
		WalletID:       w.ID,
		Type:           txType,
		Amount:         float64(amount),
		BalanceAfter:   newBalance,
		IdempotencyKey: idempotencyKey,
	}

	return s.repo.CreateTransaction(ctx, tx, txRecord)
}

func (s *WalletServiceImpl) RefundTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, amount int64, description string) error {
	if amount <= 0 {
		return errors.New("refund amount must be greater than 0")
	}

	w, err := s.repo.GetWalletByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		return err
	}

	newBalance := w.Balance + float64(amount)
	err = s.repo.UpdateWalletBalance(ctx, tx, w.ID, newBalance)
	if err != nil {
		return err
	}

	txRecord := &model.WalletTransaction{
		WalletID:     w.ID,
		Type:         "refund",
		Amount:       float64(amount),
		BalanceAfter: newBalance,
	}

	return s.repo.CreateTransaction(ctx, tx, txRecord)
}


func toTransactionResponse(t *model.WalletTransaction) *dto.TransactionResponse {
	var refStr *string
	if t.ReferenceID != nil {
		s := t.ReferenceID.String()
		refStr = &s
	}
	return &dto.TransactionResponse{
		ID:           t.ID.String(),
		Type:         t.Type,
		Amount:       t.Amount,
		ReferenceID:  refStr,
		BalanceAfter: t.BalanceAfter,
		CreatedAt:    t.CreatedAt.Format(time.RFC3339),
	}
}
