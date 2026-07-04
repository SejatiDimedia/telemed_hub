package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/model"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, cons *model.Consultation) error {
	if cons.ID == uuid.Nil {
		cons.ID = uuid.New()
	}
	cons.CreatedAt = time.Now().UTC()
	cons.UpdatedAt = cons.CreatedAt

	query := `
		INSERT INTO consultations (id, appointment_id, status, notes, started_at, ended_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Exec(ctx, query, cons.ID, cons.AppointmentID, cons.Status, cons.Notes, cons.StartedAt, cons.EndedAt, cons.CreatedAt, cons.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert consultation: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Consultation, error) {
	query := `
		SELECT id, appointment_id, status, notes, started_at, ended_at, created_at, updated_at
		FROM consultations
		WHERE id = $1`

	var c model.Consultation
	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.AppointmentID, &c.Status, &c.Notes, &c.StartedAt, &c.EndedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConsultationNotFound
		}
		return nil, fmt.Errorf("failed to fetch consultation by ID: %w", err)
	}

	return &c, nil
}

func (r *PostgresRepository) GetByAppointmentID(ctx context.Context, aptID uuid.UUID) (*model.Consultation, error) {
	query := `
		SELECT id, appointment_id, status, notes, started_at, ended_at, created_at, updated_at
		FROM consultations
		WHERE appointment_id = $1`

	var c model.Consultation
	err := r.db.QueryRow(ctx, query, aptID).Scan(
		&c.ID, &c.AppointmentID, &c.Status, &c.Notes, &c.StartedAt, &c.EndedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConsultationNotFound
		}
		return nil, fmt.Errorf("failed to fetch consultation by appointment ID: %w", err)
	}

	return &c, nil
}

func (r *PostgresRepository) Update(ctx context.Context, cons *model.Consultation) error {
	cons.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE consultations
		SET status = $1, notes = $2, started_at = $3, ended_at = $4, updated_at = $5
		WHERE id = $6`

	_, err := r.db.Exec(ctx, query, cons.Status, cons.Notes, cons.StartedAt, cons.EndedAt, cons.UpdatedAt, cons.ID)
	if err != nil {
		return fmt.Errorf("failed to update consultation: %w", err)
	}

	return nil
}
