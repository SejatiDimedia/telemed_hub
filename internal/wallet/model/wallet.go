package model

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID        uuid.UUID
	PatientID uuid.UUID
	Balance   float64
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy *uuid.UUID
	UpdatedBy *uuid.UUID
	DeletedAt *time.Time
	DeletedBy *uuid.UUID
}

type WalletTransaction struct {
	ID             uuid.UUID
	WalletID       uuid.UUID
	Type           string // top_up, consultation_payment, order_payment, refund
	Amount         float64
	ReferenceID    *uuid.UUID
	BalanceAfter   float64
	IdempotencyKey *string
	CreatedAt      time.Time
}
