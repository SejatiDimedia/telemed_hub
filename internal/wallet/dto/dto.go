package dto

type TopUpRequest struct {
	Amount float64 `json:"amount"`
}

type WalletResponse struct {
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

type TransactionResponse struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Amount       float64  `json:"amount"`
	ReferenceID  *string  `json:"reference_id,omitempty"`
	BalanceAfter float64  `json:"balance_after"`
	CreatedAt    string   `json:"created_at"`
}

type TopUpMidtransResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}
