package validator

import (
	"errors"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/dto"
)

var (
	ErrInvalidPrescriptionID = errors.New("invalid prescription UUID format")
	ErrInvalidStatus         = errors.New("status must be pending, processing, shipped, delivered, or cancelled")
)

func ValidateCreateOrder(req dto.CreateOrderRequest) error {
	if _, err := uuid.Parse(req.PrescriptionID); err != nil {
		return ErrInvalidPrescriptionID
	}
	return nil
}

func ValidateUpdateStatus(req dto.UpdateOrderStatusRequest) error {
	valid := map[string]bool{
		"pending":    true,
		"processing": true,
		"shipped":    true,
		"delivered":  true,
		"cancelled":  true,
	}
	if !valid[req.Status] {
		return ErrInvalidStatus
	}
	return nil
}
