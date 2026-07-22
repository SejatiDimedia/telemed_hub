package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
	"github.com/jackc/pgx/v5"
)

type MockWalletService struct {
	mock.Mock
}

func (m *MockWalletService) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWalletService) Deduct(ctx context.Context, userID uuid.UUID, amount int64, description string) error {
	args := m.Called(ctx, userID, amount, description)
	return args.Error(0)
}

func (m *MockWalletService) Refund(ctx context.Context, userID uuid.UUID, amount int64, description string) error {
	args := m.Called(ctx, userID, amount, description)
	return args.Error(0)
}

func (m *MockWalletService) GetBalanceDetails(ctx context.Context, userID uuid.UUID) (*dto.WalletResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.WalletResponse), args.Error(1)
}

func (m *MockWalletService) TopUp(ctx context.Context, userID uuid.UUID, amount float64, idempotencyKey *string) (*dto.TopUpMidtransResponse, error) {
	args := m.Called(ctx, userID, amount, idempotencyKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TopUpMidtransResponse), args.Error(1)
}

func (m *MockWalletService) ProcessMidtransWebhook(ctx context.Context, payload map[string]interface{}) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

func (m *MockWalletService) ListTransactions(ctx context.Context, userID uuid.UUID, typeFilter *string, page, limit int) ([]*dto.TransactionResponse, int, error) {
	args := m.Called(ctx, userID, typeFilter, page, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*dto.TransactionResponse), args.Int(1), args.Error(2)
}

func (m *MockWalletService) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*dto.TransactionResponse, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TransactionResponse), args.Error(1)
}

func (m *MockWalletService) DeductTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, amount int64, description string, idempotencyKey *string) error {
	// Dummy, not called by handler directly
	return nil
}

func (m *MockWalletService) RefundTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, amount int64, description string) error {
	// Dummy, not called by handler directly
	return nil
}

var _ service.WalletService = (*MockWalletService)(nil)

func newTestHandler(svc service.WalletService) (*WalletHandler, chi.Router) {
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewWalletHandler(svc, cfg, nil, log)

	r := chi.NewRouter()
	r.Get("/wallet", h.GetBalance)
	r.Post("/wallet/top-up", h.TopUp)
	r.Get("/wallet/transactions", h.ListTransactions)

	return h, r
}

func withAuth(req *http.Request, userID uuid.UUID, roles []string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, middleware.RolesContextKey, roles)
	return req.WithContext(ctx)
}

func TestWalletHandler_GetBalance_Success(t *testing.T) {
	mockSvc := new(MockWalletService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	expectedResp := &dto.WalletResponse{
		Balance:  150000.0,
		Currency: "IDR",
	}

	mockSvc.On("GetBalanceDetails", mock.Anything, userID).Return(expectedResp, nil).Once()

	req, _ := http.NewRequest(http.MethodGet, "/wallet", nil)
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var envelope struct {
		Success bool               `json:"success"`
		Data    dto.WalletResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&envelope))
	assert.True(t, envelope.Success)
	assert.Equal(t, 150000.0, envelope.Data.Balance)
	mockSvc.AssertExpectations(t)
}

func TestWalletHandler_TopUp_Success(t *testing.T) {
	mockSvc := new(MockWalletService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	reqBody := dto.TopUpRequest{Amount: 50000.0}

	expectedResp := &dto.TopUpMidtransResponse{
		Token:       "snap-token-123",
		RedirectURL: "https://app.sandbox.midtrans.com/snap/v2/vtweb/snap-token-123",
	}

	mockSvc.On("TopUp", mock.Anything, userID, 50000.0, mock.Anything).Return(expectedResp, nil).Once()

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/wallet/top-up", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-key-1")
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	mockSvc.AssertExpectations(t)
}
