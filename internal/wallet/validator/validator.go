package validator

import (
	"errors"

	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/dto"
)

var (
	ErrInvalidAmount = errors.New("top-up amount must be greater than 0")
)

func ValidateTopUp(req dto.TopUpRequest) error {
	if req.Amount <= 0 {
		return ErrInvalidAmount
	}
	return nil
}
