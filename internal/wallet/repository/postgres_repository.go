package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/model"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) getPatientIDByUserID(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (uuid.UUID, error) {
	query := `SELECT id FROM patients WHERE user_id = $1`
	var patientID uuid.UUID
	var err error

	if tx != nil {
		err = tx.QueryRow(ctx, query, userID).Scan(&patientID)
	} else {
		err = r.db.QueryRow(ctx, query, userID).Scan(&patientID)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrWalletNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to map user to patient: %w", err)
	}
	return patientID, nil
}

func (r *PostgresRepository) GetOrCreateWallet(ctx context.Context, tx pgx.Tx, patientID uuid.UUID) (*model.Wallet, error) {
	selectQuery := `
		SELECT id, patient_id, balance, created_at, updated_at
		FROM wallets
		WHERE patient_id = $1 AND deleted_at IS NULL
	`

	var w model.Wallet
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, selectQuery, patientID).Scan(&w.ID, &w.PatientID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	} else {
		err = r.db.QueryRow(ctx, selectQuery, patientID).Scan(&w.ID, &w.PatientID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	}

	if err == nil {
		return &w, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to select wallet: %w", err)
	}

	// Create a new wallet
	w.ID = uuid.New()
	w.PatientID = patientID
	w.Balance = 0.0
	w.CreatedAt = time.Now().UTC()
	w.UpdatedAt = w.CreatedAt

	insertQuery := `
		INSERT INTO wallets (id, patient_id, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	if tx != nil {
		_, err = tx.Exec(ctx, insertQuery, w.ID, w.PatientID, w.Balance, w.CreatedAt, w.UpdatedAt)
	} else {
		_, err = r.db.Exec(ctx, insertQuery, w.ID, w.PatientID, w.Balance, w.CreatedAt, w.UpdatedAt)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to insert wallet: %w", err)
	}

	return &w, nil
}

func (r *PostgresRepository) GetWalletByUserID(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Wallet, error) {
	patientID, err := r.getPatientIDByUserID(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	return r.GetOrCreateWallet(ctx, tx, patientID)
}

func (r *PostgresRepository) GetWalletByUserIDForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Wallet, error) {
	patientID, err := r.getPatientIDByUserID(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	// First ensure wallet exists
	_, err = r.GetOrCreateWallet(ctx, tx, patientID)
	if err != nil {
		return nil, err
	}

	// Lock the wallet row
	query := `
		SELECT id, patient_id, balance, created_at, updated_at
		FROM wallets
		WHERE patient_id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`
	var w model.Wallet
	if tx != nil {
		err = tx.QueryRow(ctx, query, patientID).Scan(&w.ID, &w.PatientID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	} else {
		err = r.db.QueryRow(ctx, query, patientID).Scan(&w.ID, &w.PatientID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to fetch wallet for update: %w", err)
	}

	return &w, nil
}

func (r *PostgresRepository) GetWalletByIDForUpdate(ctx context.Context, tx pgx.Tx, walletID uuid.UUID) (*model.Wallet, error) {
	query := `
		SELECT id, patient_id, balance, created_at, updated_at
		FROM wallets
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`
	var w model.Wallet
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, walletID).Scan(&w.ID, &w.PatientID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	} else {
		err = r.db.QueryRow(ctx, query, walletID).Scan(&w.ID, &w.PatientID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to fetch wallet for update: %w", err)
	}

	return &w, nil
}

func (r *PostgresRepository) UpdateWalletBalance(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, balance float64) error {
	now := time.Now().UTC()
	query := `
		UPDATE wallets
		SET balance = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`
	var tag tagWrapper
	var err error

	if tx != nil {
		tag, err = tx.Exec(ctx, query, balance, now, walletID)
	} else {
		tag, err = r.db.Exec(ctx, query, balance, now, walletID)
	}

	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWalletNotFound
	}
	return nil
}

type tagWrapper interface {
	RowsAffected() int64
}

func (r *PostgresRepository) CreateTransaction(ctx context.Context, tx pgx.Tx, txRecord *model.WalletTransaction) error {
	if txRecord.ID == uuid.Nil {
		txRecord.ID = uuid.New()
	}
	txRecord.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, reference_id, balance_after, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query,
			txRecord.ID, txRecord.WalletID, txRecord.Type, txRecord.Amount,
			txRecord.ReferenceID, txRecord.BalanceAfter, txRecord.IdempotencyKey, txRecord.CreatedAt,
		)
	} else {
		_, err = r.db.Exec(ctx, query,
			txRecord.ID, txRecord.WalletID, txRecord.Type, txRecord.Amount,
			txRecord.ReferenceID, txRecord.BalanceAfter, txRecord.IdempotencyKey, txRecord.CreatedAt,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to create wallet transaction ledger record: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetTransactionByIdempotencyKey(ctx context.Context, tx pgx.Tx, key string) (*model.WalletTransaction, error) {
	query := `
		SELECT id, wallet_id, type, amount, reference_id, balance_after, idempotency_key, created_at
		FROM wallet_transactions
		WHERE idempotency_key = $1
	`
	var t model.WalletTransaction
	var err error

	if tx != nil {
		err = tx.QueryRow(ctx, query, key).Scan(
			&t.ID, &t.WalletID, &t.Type, &t.Amount, &t.ReferenceID, &t.BalanceAfter, &t.IdempotencyKey, &t.CreatedAt,
		)
	} else {
		err = r.db.QueryRow(ctx, query, key).Scan(
			&t.ID, &t.WalletID, &t.Type, &t.Amount, &t.ReferenceID, &t.BalanceAfter, &t.IdempotencyKey, &t.CreatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil when idempotency key is not found
		}
		return nil, fmt.Errorf("failed to get transaction by idempotency key: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) ListTransactions(ctx context.Context, walletID uuid.UUID, typeFilter *string, page, limit int) ([]*model.WalletTransaction, int, error) {
	var whereClauses []string
	var args []any
	argCount := 1

	whereClauses = append(whereClauses, fmt.Sprintf("wallet_id = $%d", argCount))
	args = append(args, walletID)
	argCount++

	if typeFilter != nil && *typeFilter != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("type = $%d", argCount))
		args = append(args, *typeFilter)
		argCount++
	}

	whereSQL := "WHERE " + strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM wallet_transactions %s", whereSQL)
	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	if total == 0 {
		return []*model.WalletTransaction{}, 0, nil
	}

	offset := (page - 1) * limit
	listQuery := fmt.Sprintf(`
		SELECT id, wallet_id, type, amount, reference_id, balance_after, idempotency_key, created_at
		FROM wallet_transactions
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argCount, argCount+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var txs []*model.WalletTransaction
	for rows.Next() {
		var t model.WalletTransaction
		err := rows.Scan(
			&t.ID, &t.WalletID, &t.Type, &t.Amount, &t.ReferenceID, &t.BalanceAfter, &t.IdempotencyKey, &t.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan transaction: %w", err)
		}
		txs = append(txs, &t)
	}

	return txs, total, nil
}

func (r *PostgresRepository) VerifyLedgerConsistency(ctx context.Context, walletID uuid.UUID) (float64, error) {
	// Replay all transactions for the wallet to calculate total balance.
	// top_up & refund are additions (+).
	// order_payment & consultation_payment are deductions (-).
	query := `
		SELECT type, amount
		FROM wallet_transactions
		WHERE wallet_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query, walletID)
	if err != nil {
		return 0, fmt.Errorf("failed to query transactions for ledger verification: %w", err)
	}
	defer rows.Close()

	calculatedBalance := 0.0
	for rows.Next() {
		var tType string
		var amount float64
		if err := rows.Scan(&tType, &amount); err != nil {
			return 0, fmt.Errorf("failed to scan transaction row for ledger check: %w", err)
		}

		switch tType {
		case "top_up", "refund":
			calculatedBalance += amount
		case "order_payment", "consultation_payment":
			calculatedBalance -= amount
		}
	}

	return calculatedBalance, nil
}
