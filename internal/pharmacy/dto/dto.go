package dto

type CreateOrderRequest struct {
	PrescriptionID string `json:"prescription_id"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

type OrderResponse struct {
	ID             string              `json:"id"`
	PatientID      string              `json:"patient_id"`
	PrescriptionID *string             `json:"prescription_id,omitempty"`
	Status         string              `json:"status"`
	TotalAmount    float64             `json:"total_amount"`
	Items          []OrderItemResponse `json:"items"`
	CreatedAt      string              `json:"created_at"`
	UpdatedAt      string              `json:"updated_at"`
}

type OrderItemResponse struct {
	ID         string  `json:"id"`
	MedicineID string  `json:"medicine_id"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
}
