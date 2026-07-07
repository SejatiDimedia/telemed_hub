package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/dto"
)

var (
	ErrUnauthorized     = errors.New("unauthorized action on inventory")
	ErrMedicineNotFound = errors.New("medicine not found")
	ErrOutOfStock       = errors.New("medicine out of stock")
)

type InventoryService interface {
	Create(ctx context.Context, adminUserID uuid.UUID, req dto.CreateMedicineRequest) (*dto.MedicineResponse, error)
	Update(ctx context.Context, adminUserID uuid.UUID, id uuid.UUID, req dto.UpdateMedicineRequest) (*dto.MedicineResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.MedicineResponse, error)
	List(ctx context.Context, nameFilter *string, reqPrescFilter *bool, page, limit int) ([]*dto.MedicineResponse, int, error)
	Delete(ctx context.Context, adminUserID uuid.UUID, id uuid.UUID) error

	// Concurrency & Stock management methods for cross-module atomic transactions
	DecrementStock(ctx context.Context, tx pgx.Tx, medicineID uuid.UUID, quantity int) error
	IncrementStock(ctx context.Context, tx pgx.Tx, medicineID uuid.UUID, quantity int) error
}
