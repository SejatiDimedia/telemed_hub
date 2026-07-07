package model

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID             uuid.UUID
	PatientID      uuid.UUID
	PrescriptionID *uuid.UUID
	Status         string // pending, processing, shipped, delivered, cancelled
	TotalAmount    float64
	Items          []OrderItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      *uuid.UUID
	UpdatedBy      *uuid.UUID
	DeletedAt      *time.Time
	DeletedBy      *uuid.UUID
}

type OrderItem struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	MedicineID uuid.UUID
	Quantity   int
	UnitPrice  float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
