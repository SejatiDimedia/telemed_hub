package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timurdianradhasejati/telemed_hub/internal/patient/model"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Patient, error) {
	query := `
		SELECT p.id, p.user_id, u.email, u.full_name, u.phone_number, p.date_of_birth, p.gender, p.blood_type, p.created_at, p.updated_at, p.deleted_at
		FROM patients p
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id = $1 AND p.deleted_at IS NULL AND u.deleted_at IS NULL`

	var p model.Patient
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&p.ID, &p.UserID, &p.Email, &p.FullName, &p.PhoneNumber, &p.DateOfBirth, &p.Gender, &p.BloodType, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPatientNotFound
		}
		return nil, fmt.Errorf("failed to fetch patient by user ID: %w", err)
	}

	return &p, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Patient, error) {
	query := `
		SELECT p.id, p.user_id, u.email, u.full_name, u.phone_number, p.date_of_birth, p.gender, p.blood_type, p.created_at, p.updated_at, p.deleted_at
		FROM patients p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = $1 AND p.deleted_at IS NULL AND u.deleted_at IS NULL`

	var p model.Patient
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.UserID, &p.Email, &p.FullName, &p.PhoneNumber, &p.DateOfBirth, &p.Gender, &p.BloodType, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPatientNotFound
		}
		return nil, fmt.Errorf("failed to fetch patient by ID: %w", err)
	}

	return &p, nil
}

func (r *PostgresRepository) Update(ctx context.Context, p *model.Patient) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	now := time.Now().UTC()

	// 1. Update shared user fields
	queryUser := `
		UPDATE users
		SET phone_number = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL`

	res, err := tx.Exec(ctx, queryUser, p.PhoneNumber, now, p.UserID)
	if err != nil {
		return fmt.Errorf("failed to update user fields: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrPatientNotFound
	}

	// 2. Update patient profile fields
	queryPatient := `
		UPDATE patients
		SET date_of_birth = $1, gender = $2, blood_type = $3, updated_at = $4
		WHERE id = $5 AND deleted_at IS NULL`

	res, err = tx.Exec(ctx, queryPatient, p.DateOfBirth, p.Gender, p.BloodType, now, p.ID)
	if err != nil {
		return fmt.Errorf("failed to update patient fields: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrPatientNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	p.UpdatedAt = now
	return nil
}
