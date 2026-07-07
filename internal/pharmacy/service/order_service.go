package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	inventorySvc "github.com/timurdianradhasejati/telemed_hub/internal/inventory/service"
	patientSvc "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	prescriptionSvc "github.com/timurdianradhasejati/telemed_hub/internal/prescription/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/repository"
	walletSvc "github.com/timurdianradhasejati/telemed_hub/internal/wallet"
)

var (
	ErrInsufficientBalance    = errors.New("insufficient wallet balance")
	ErrOutOfStock              = errors.New("medicine out of stock")
	ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled after transitioning to processing")
)

type OrderServiceImpl struct {
	repo            repository.OrderRepository
	db              *pgxpool.Pool
	prescriptionSvc prescriptionSvc.PrescriptionService
	inventorySvc    inventorySvc.InventoryService
	patientSvc      patientSvc.PatientService
	walletSvc       walletSvc.WalletService
}

func NewOrderService(
	repo repository.OrderRepository,
	db *pgxpool.Pool,
	prescriptionSvc prescriptionSvc.PrescriptionService,
	inventorySvc inventorySvc.InventoryService,
	patientSvc patientSvc.PatientService,
	walletSvc walletSvc.WalletService,
) *OrderServiceImpl {
	return &OrderServiceImpl{
		repo:            repo,
		db:              db,
		prescriptionSvc: prescriptionSvc,
		inventorySvc:    inventorySvc,
		patientSvc:      patientSvc,
		walletSvc:       walletSvc,
	}
}

func (s *OrderServiceImpl) Create(ctx context.Context, patientUserID uuid.UUID, req dto.CreateOrderRequest, idempotencyKey *string) (*dto.OrderResponse, error) {
	presID, err := uuid.Parse(req.PrescriptionID)
	if err != nil {
		return nil, fmt.Errorf("invalid prescription ID: %w", err)
	}

	// 1. Check Idempotency
	if idempotencyKey != nil && *idempotencyKey != "" {
		existingTx, err := s.walletSvc.GetTransactionByIdempotencyKey(ctx, *idempotencyKey)
		if err == nil && existingTx != nil && existingTx.ReferenceID != nil {
			orderID, err := uuid.Parse(*existingTx.ReferenceID)
			if err == nil {
				ord, err := s.repo.GetByID(ctx, orderID)
				if err == nil {
					return toResponse(ord), nil
				}
			}
		}
	}

	// 2. Fetch prescription and verify patient owns it
	pres, err := s.prescriptionSvc.GetByID(ctx, presID, patientUserID, []string{"patient"})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prescription: %w", err)
	}

	// 3. Resolve Patient ID
	patProfile, err := s.patientSvc.GetProfileByUserID(ctx, patientUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve patient profile: %w", err)
	}
	patientID, _ := uuid.Parse(patProfile.ID)

	// 4. Begin Database Transaction for Atomic Stock and Wallet Deduction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	orderID := uuid.New()
	var totalAmount float64
	var orderItems []model.OrderItem

	// 5. Concurrency Control: Decrement Medicine Stock using SELECT FOR UPDATE inside tx
	for _, item := range pres.Items {
		medID, _ := uuid.Parse(item.MedicineID)

		// Check medicine price & catalog requirements
		med, err := s.inventorySvc.GetByID(ctx, medID)
		if err != nil {
			return nil, fmt.Errorf("failed to check medicine: %w", err)
		}

		// Perform transactional stock decrement (pessimistic lock)
		err = s.inventorySvc.DecrementStock(ctx, tx, medID, item.Quantity)
		if err != nil {
			if errors.Is(err, inventorySvc.ErrOutOfStock) {
				return nil, ErrOutOfStock
			}
			return nil, err
		}

		orderItems = append(orderItems, model.OrderItem{
			MedicineID: medID,
			Quantity:   item.Quantity,
			UnitPrice:  med.UnitPrice,
		})

		totalAmount += med.UnitPrice * float64(item.Quantity)
	}

	// 6. Transactional Wallet Deduction
	err = s.walletSvc.DeductTx(ctx, tx, patientUserID, int64(totalAmount), fmt.Sprintf("Payment for pharmacy order %s", orderID), idempotencyKey)
	if err != nil {
		if errors.Is(err, walletSvc.ErrInsufficientBalance) || strings.Contains(err.Error(), "insufficient") {
			return nil, ErrInsufficientBalance
		}
		return nil, err
	}

	// 7. Persist Order and Order Items
	ord := &model.Order{
		ID:             orderID,
		PatientID:      patientID,
		PrescriptionID: &presID,
		Status:         "pending",
		TotalAmount:    totalAmount,
		Items:          orderItems,
		CreatedBy:      &patientUserID,
	}

	err = s.repo.Create(ctx, tx, ord)
	if err != nil {
		return nil, fmt.Errorf("failed to persist order: %w", err)
	}

	// 8. Commit Transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit order transaction: %w", err)
	}

	// Fetch saved order with actual timestamps
	saved, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload order: %w", err)
	}

	return toResponse(saved), nil
}

func (s *OrderServiceImpl) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.OrderResponse, error) {
	ord, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	// Check Authorization
	if hasRole(roles, "admin") || hasRole(roles, "pharmacy_staff") {
		return toResponse(ord), nil
	}

	if hasRole(roles, "patient") {
		patProfile, err := s.patientSvc.GetProfileByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve patient profile: %w", err)
		}
		patientID, _ := uuid.Parse(patProfile.ID)
		if ord.PatientID != patientID {
			return nil, ErrUnauthorized
		}
		return toResponse(ord), nil
	}

	return nil, ErrUnauthorized
}

func (s *OrderServiceImpl) List(ctx context.Context, userID uuid.UUID, roles []string, statusFilter string) ([]*dto.OrderResponse, error) {
	var patientIDPtr *uuid.UUID
	var statusPtr *string
	if statusFilter != "" {
		statusPtr = &statusFilter
	}

	if hasRole(roles, "patient") {
		patProfile, err := s.patientSvc.GetProfileByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve patient profile: %w", err)
		}
		patientID, _ := uuid.Parse(patProfile.ID)
		patientIDPtr = &patientID
	}

	// We pass hardcoded pagination limit 100 for global listing page
	orders, _, err := s.repo.List(ctx, patientIDPtr, statusPtr, 1, 100)
	if err != nil {
		return nil, err
	}

	respList := make([]*dto.OrderResponse, 0, len(orders))
	for _, o := range orders {
		respList = append(respList, toResponse(o))
	}
	return respList, nil
}

func (s *OrderServiceImpl) UpdateStatus(ctx context.Context, adminOrStaffUserID uuid.UUID, id uuid.UUID, req dto.UpdateOrderStatusRequest) (*dto.OrderResponse, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	ord, err := s.repo.GetByIDForUpdate(ctx, tx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	// Enforce order status transitions rules: pending -> processing -> shipped -> delivered
	if !isValidTransition(ord.Status, req.Status) {
		return nil, ErrInvalidStatusTransition
	}

	// Handle Refund and Stock release side effect if transitioning to cancelled
	if req.Status == "cancelled" {
		err := s.processOrderCancellationAndRefund(ctx, tx, ord)
		if err != nil {
			return nil, err
		}
	}

	err = s.repo.UpdateStatus(ctx, tx, id, req.Status)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit update status transaction: %w", err)
	}

	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return toResponse(updated), nil
}

func (s *OrderServiceImpl) Cancel(ctx context.Context, patientUserID uuid.UUID, id uuid.UUID) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	ord, err := s.repo.GetByIDForUpdate(ctx, tx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return ErrOrderNotFound
		}
		return err
	}

	// Verify caller owns the order
	patProfile, err := s.patientSvc.GetProfileByUserID(ctx, patientUserID)
	if err != nil {
		return fmt.Errorf("failed to resolve patient profile: %w", err)
	}
	patientID, _ := uuid.Parse(patProfile.ID)
	if ord.PatientID != patientID {
		return ErrUnauthorized
	}

	// Patients can only cancel when order is still 'pending' (before processing)
	if ord.Status != "pending" {
		return ErrOrderCannotBeCancelled
	}

	err = s.processOrderCancellationAndRefund(ctx, tx, ord)
	if err != nil {
		return err
	}

	err = s.repo.UpdateStatus(ctx, tx, id, "cancelled")
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *OrderServiceImpl) processOrderCancellationAndRefund(ctx context.Context, tx pgx.Tx, ord *model.Order) error {
	// 1. Release Stock back to catalog
	for _, item := range ord.Items {
		err := s.inventorySvc.IncrementStock(ctx, tx, item.MedicineID, item.Quantity)
		if err != nil {
			return fmt.Errorf("failed to release stock back to inventory: %w", err)
		}
	}

	// 2. Resolve Patient UserID to refund
	var userID uuid.UUID
	err := s.db.QueryRow(ctx, `SELECT user_id FROM patients WHERE id = $1`, ord.PatientID).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to resolve user ID for patient: %w", err)
	}

	// 3. Perform refund
	err = s.walletSvc.RefundTx(ctx, tx, userID, int64(ord.TotalAmount), fmt.Sprintf("Refund for cancelled order %s", ord.ID))
	if err != nil {
		return fmt.Errorf("failed to refund patient wallet: %w", err)
	}

	return nil
}

func isValidTransition(current, next string) bool {
	if current == next {
		return true
	}
	switch current {
	case "pending":
		return next == "processing" || next == "cancelled"
	case "processing":
		return next == "shipped"
	case "shipped":
		return next == "delivered"
	}
	return false
}

func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

func toResponse(o *model.Order) *dto.OrderResponse {
	var prescStr *string
	if o.PrescriptionID != nil {
		s := o.PrescriptionID.String()
		prescStr = &s
	}

	items := make([]dto.OrderItemResponse, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, dto.OrderItemResponse{
			ID:         it.ID.String(),
			MedicineID: it.MedicineID.String(),
			Quantity:   it.Quantity,
			UnitPrice:  it.UnitPrice,
		})
	}

	return &dto.OrderResponse{
		ID:             o.ID.String(),
		PatientID:      o.PatientID.String(),
		PrescriptionID: prescStr,
		Status:         o.Status,
		TotalAmount:    o.TotalAmount,
		Items:          items,
		CreatedAt:      o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      o.UpdatedAt.Format(time.RFC3339),
	}
}
