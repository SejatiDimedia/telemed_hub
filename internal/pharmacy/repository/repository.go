package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/model"
)

var (
	ErrOrderNotFound = errors.New("order not found")
)

type OrderRepository interface {
	Create(ctx context.Context, tx pgx.Tx, ord *model.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
	GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error)
	UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error
	List(ctx context.Context, patientID *uuid.UUID, statusFilter *string, page, limit int) ([]*model.Order, int, error)
}
