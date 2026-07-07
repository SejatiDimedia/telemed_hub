package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/dto"
)

var (
	ErrUnauthorized            = errors.New("unauthorized action on orders")
	ErrOrderNotFound           = errors.New("order not found")
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
)

type OrderService interface {
	Create(ctx context.Context, patientUserID uuid.UUID, req dto.CreateOrderRequest, idempotencyKey *string) (*dto.OrderResponse, error)
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.OrderResponse, error)
	List(ctx context.Context, userID uuid.UUID, roles []string, statusFilter string) ([]*dto.OrderResponse, error)
	UpdateStatus(ctx context.Context, adminOrStaffUserID uuid.UUID, id uuid.UUID, req dto.UpdateOrderStatusRequest) (*dto.OrderResponse, error)
	Cancel(ctx context.Context, patientUserID uuid.UUID, id uuid.UUID) error
}
