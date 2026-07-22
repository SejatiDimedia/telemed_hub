package service

import (
	"context"
	"crypto/sha512"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
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
	cfg      *config.Config
}

func NewWalletService(repo repository.WalletRepository, db *pgxpool.Pool, cfg *config.Config) *WalletServiceImpl {
	// Initialize Midtrans
	midtrans.ServerKey = cfg.Midtrans.ServerKey
	if cfg.Midtrans.Environment == "production" {
		midtrans.Environment = midtrans.Production
	} else {
		midtrans.Environment = midtrans.Sandbox
	}

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
		cfg:      cfg,
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

func (s *WalletServiceImpl) TopUp(ctx context.Context, userID uuid.UUID, amount float64, idempotencyKey *string) (*dto.TopUpMidtransResponse, error) {
	if amount <= 0 {
		return nil, errors.New("top-up amount must be greater than 0")
	}
	if amount > s.maxTopUp {
		return nil, ErrMaxTopUpExceeded
	}

	_, err := s.repo.GetWalletByUserID(ctx, nil, userID)
	if err != nil {
		return nil, err
	}

	// Generate a unique order ID for Midtrans (Max 50 chars)
	// Format: TU-<userID>-<short_random>
	// UUID = 36 chars, "TU-" = 3, "-" = 1, random = 8 → total ~48 chars (under 50)
	shortRandom := uuid.New().String()[:8]
	orderID := fmt.Sprintf("TU-%s-%s", userID.String(), shortRandom)

	// Determine frontend base URL for redirect after payment
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: int64(amount),
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: "User",
			LName: userID.String(),
		},
		Callbacks: &snap.Callbacks{
			Finish: fmt.Sprintf("%s/patient/wallet?payment=success", frontendURL),
		},
	}

	snapResp, midErr := snap.CreateTransaction(req)
	if midErr != nil {
		return nil, fmt.Errorf("failed to create midtrans transaction: %s", midErr.Message)
	}

	return &dto.TopUpMidtransResponse{
		Token:       snapResp.Token,
		RedirectURL: snapResp.RedirectURL,
	}, nil
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
		return nil, repository.ErrTransactionNotFound
	}

	return toTransactionResponse(t), nil
}

func (s *WalletServiceImpl) ProcessMidtransWebhook(ctx context.Context, payload map[string]interface{}) error {
	orderID, ok := payload["order_id"].(string)
	if !ok {
		return errors.New("missing order_id in webhook payload")
	}

	transactionStatus, ok := payload["transaction_status"].(string)
	if !ok {
		return errors.New("missing transaction_status in webhook payload")
	}

	statusCode, ok := payload["status_code"].(string)
	if !ok {
		return errors.New("missing status_code in webhook payload")
	}

	grossAmountStr, ok := payload["gross_amount"].(string)
	if !ok {
		return errors.New("missing gross_amount in webhook payload")
	}

	signatureKey, ok := payload["signature_key"].(string)
	if !ok {
		return errors.New("missing signature_key in webhook payload")
	}

	// Verify signature key
	// SHA512(order_id + status_code + gross_amount + server_key)
	hashInput := orderID + statusCode + grossAmountStr + s.cfg.Midtrans.ServerKey
	expectedSignature := fmt.Sprintf("%x", sha512.Sum512([]byte(hashInput)))

	if signatureKey != expectedSignature {
		return errors.New("invalid signature key")
	}

	// Process only settled/captured transactions
	if transactionStatus == "settlement" || transactionStatus == "capture" {
		// Parse orderID: TU-<userID>-<random>
		// userID is a UUID (36 chars) at positions after "TU-"
		if !strings.HasPrefix(orderID, "TU-") {
			return errors.New("invalid order_id format: missing TU- prefix")
		}
		trimmed := strings.TrimPrefix(orderID, "TU-")
		// The userID UUID is the first 36 characters
		if len(trimmed) < 36 {
			return errors.New("invalid order_id format: too short for userID")
		}
		userIDStr := trimmed[:36]
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return fmt.Errorf("invalid user_id in order_id: %w", err)
		}

		amount, err := strconv.ParseFloat(grossAmountStr, 64)
		if err != nil {
			return fmt.Errorf("invalid gross_amount format: %w", err)
		}

		// Idempotency check: see if this orderID was already processed as an idempotency key
		tx, err := s.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		existing, err := s.repo.GetTransactionByIdempotencyKey(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if existing != nil {
			// Already processed
			return nil
		}

		// Lock and update wallet by userID
		w, err := s.repo.GetWalletByUserIDForUpdate(ctx, tx, userID)
		if err != nil {
			return err
		}

		newBalance := w.Balance + amount
		err = s.repo.UpdateWalletBalance(ctx, tx, w.ID, newBalance)
		if err != nil {
			return err
		}

		txRecord := &model.WalletTransaction{
			WalletID:       w.ID,
			Type:           "top_up",
			Amount:         amount,
			BalanceAfter:   newBalance,
			IdempotencyKey: &orderID,
		}

		err = s.repo.CreateTransaction(ctx, tx, txRecord)
		if err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
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
