package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/model"
)

var (
	ErrMedicineNotFound = errors.New("medicine not found")
)

type MedicineRepository interface {
	Create(ctx context.Context, med *model.Medicine) error
	Update(ctx context.Context, med *model.Medicine) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Medicine, error)
	GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Medicine, error)
	List(ctx context.Context, nameFilter *string, reqPrescFilter *bool, page, limit int) ([]*model.Medicine, int, error)
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	UpdateStock(ctx context.Context, tx pgx.Tx, id uuid.UUID, newStock int) error
}
