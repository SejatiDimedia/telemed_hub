package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/repository"
)

type InventoryServiceImpl struct {
	repo repository.MedicineRepository
}

func NewInventoryService(repo repository.MedicineRepository) *InventoryServiceImpl {
	return &InventoryServiceImpl{repo: repo}
}

func (s *InventoryServiceImpl) Create(ctx context.Context, adminUserID uuid.UUID, req dto.CreateMedicineRequest) (*dto.MedicineResponse, error) {
	med := &model.Medicine{
		Name:                 req.Name,
		UnitPrice:            req.UnitPrice,
		StockQuantity:        req.StockQuantity,
		RequiresPrescription: req.RequiresPrescription,
		CreatedBy:            &adminUserID,
	}

	if err := s.repo.Create(ctx, med); err != nil {
		return nil, err
	}

	return toResponse(med), nil
}

func (s *InventoryServiceImpl) Update(ctx context.Context, adminUserID uuid.UUID, id uuid.UUID, req dto.UpdateMedicineRequest) (*dto.MedicineResponse, error) {
	med, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrMedicineNotFound) {
			return nil, ErrMedicineNotFound
		}
		return nil, err
	}

	med.Name = req.Name
	med.UnitPrice = req.UnitPrice
	med.StockQuantity = req.StockQuantity
	med.RequiresPrescription = req.RequiresPrescription
	med.UpdatedBy = &adminUserID

	if err := s.repo.Update(ctx, med); err != nil {
		if errors.Is(err, repository.ErrMedicineNotFound) {
			return nil, ErrMedicineNotFound
		}
		return nil, err
	}

	return toResponse(med), nil
}

func (s *InventoryServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*dto.MedicineResponse, error) {
	med, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrMedicineNotFound) {
			return nil, ErrMedicineNotFound
		}
		return nil, err
	}
	return toResponse(med), nil
}

func (s *InventoryServiceImpl) List(ctx context.Context, nameFilter *string, reqPrescFilter *bool, page, limit int) ([]*dto.MedicineResponse, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	medicines, total, err := s.repo.List(ctx, nameFilter, reqPrescFilter, page, limit)
	if err != nil {
		return nil, 0, err
	}

	respList := make([]*dto.MedicineResponse, 0, len(medicines))
	for _, m := range medicines {
		respList = append(respList, toResponse(m))
	}
	return respList, total, nil
}

func (s *InventoryServiceImpl) Delete(ctx context.Context, adminUserID uuid.UUID, id uuid.UUID) error {
	err := s.repo.SoftDelete(ctx, id, adminUserID)
	if err != nil {
		if errors.Is(err, repository.ErrMedicineNotFound) {
			return ErrMedicineNotFound
		}
		return err
	}
	return nil
}

func toResponse(m *model.Medicine) *dto.MedicineResponse {
	return &dto.MedicineResponse{
		ID:                   m.ID.String(),
		Name:                 m.Name,
		UnitPrice:            m.UnitPrice,
		StockQuantity:        m.StockQuantity,
		RequiresPrescription: m.RequiresPrescription,
		CreatedAt:            m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            m.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *InventoryServiceImpl) DecrementStock(ctx context.Context, tx pgx.Tx, medicineID uuid.UUID, quantity int) error {
	m, err := s.repo.GetByIDForUpdate(ctx, tx, medicineID)
	if err != nil {
		if errors.Is(err, repository.ErrMedicineNotFound) {
			return ErrMedicineNotFound
		}
		return err
	}

	if m.StockQuantity < quantity {
		return ErrOutOfStock
	}

	return s.repo.UpdateStock(ctx, tx, medicineID, m.StockQuantity-quantity)
}

func (s *InventoryServiceImpl) IncrementStock(ctx context.Context, tx pgx.Tx, medicineID uuid.UUID, quantity int) error {
	m, err := s.repo.GetByIDForUpdate(ctx, tx, medicineID)
	if err != nil {
		if errors.Is(err, repository.ErrMedicineNotFound) {
			return ErrMedicineNotFound
		}
		return err
	}

	return s.repo.UpdateStock(ctx, tx, medicineID, m.StockQuantity+quantity)
}

