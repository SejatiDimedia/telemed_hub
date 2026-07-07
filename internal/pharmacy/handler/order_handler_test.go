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
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) Create(ctx context.Context, patientUserID uuid.UUID, req dto.CreateOrderRequest, idempotencyKey *string) (*dto.OrderResponse, error) {
	args := m.Called(ctx, patientUserID, req, idempotencyKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.OrderResponse), args.Error(1)
}

func (m *MockOrderService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.OrderResponse, error) {
	args := m.Called(ctx, id, userID, roles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.OrderResponse), args.Error(1)
}

func (m *MockOrderService) List(ctx context.Context, userID uuid.UUID, roles []string, statusFilter string) ([]*dto.OrderResponse, error) {
	args := m.Called(ctx, userID, roles, statusFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*dto.OrderResponse), args.Error(1)
}

func (m *MockOrderService) UpdateStatus(ctx context.Context, adminOrStaffUserID uuid.UUID, id uuid.UUID, req dto.UpdateOrderStatusRequest) (*dto.OrderResponse, error) {
	args := m.Called(ctx, adminOrStaffUserID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.OrderResponse), args.Error(1)
}

func (m *MockOrderService) Cancel(ctx context.Context, patientUserID uuid.UUID, id uuid.UUID) error {
	args := m.Called(ctx, patientUserID, id)
	return args.Error(0)
}

var _ service.OrderService = (*MockOrderService)(nil)

func newTestHandler(svc service.OrderService) (*OrderHandler, chi.Router) {
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewOrderHandler(svc, cfg, nil, log)

	r := chi.NewRouter()
	r.Post("/orders", h.Create)
	r.Get("/orders/{id}", h.GetByID)
	r.Get("/orders", h.List)
	r.Put("/orders/{id}/status", h.UpdateStatus)
	r.Post("/orders/{id}/cancel", h.Cancel)

	return h, r
}

func withAuth(req *http.Request, userID uuid.UUID, roles []string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, middleware.RolesContextKey, roles)
	return req.WithContext(ctx)
}

func TestOrderHandler_Create_Success(t *testing.T) {
	mockSvc := new(MockOrderService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	presID := uuid.New().String()
	reqBody := dto.CreateOrderRequest{PrescriptionID: presID}

	expectedResp := &dto.OrderResponse{
		ID:          uuid.New().String(),
		PatientID:   uuid.New().String(),
		Status:      "pending",
		TotalAmount: 120000.0,
		Items:       []dto.OrderItemResponse{},
	}

	mockSvc.On("Create", mock.Anything, userID, reqBody, mock.Anything).Return(expectedResp, nil).Once()

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	mockSvc.AssertExpectations(t)
}

func TestOrderHandler_Create_ForbiddenForDoctor(t *testing.T) {
	mockSvc := new(MockOrderService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	reqBody := dto.CreateOrderRequest{PrescriptionID: uuid.New().String()}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, userID, []string{"doctor"}) // Doctor cannot make order

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockSvc.AssertNotCalled(t, "Create")
}

func TestOrderHandler_GetByID_Success(t *testing.T) {
	mockSvc := new(MockOrderService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	ordID := uuid.New()
	expectedResp := &dto.OrderResponse{
		ID:          ordID.String(),
		Status:      "pending",
		TotalAmount: 150000.0,
	}

	mockSvc.On("GetByID", mock.Anything, ordID, userID, []string{"patient"}).Return(expectedResp, nil).Once()

	req, _ := http.NewRequest(http.MethodGet, "/orders/"+ordID.String(), nil)
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}

func TestOrderHandler_UpdateStatus_ForbiddenForPatient(t *testing.T) {
	mockSvc := new(MockOrderService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	ordID := uuid.New()
	reqBody := dto.UpdateOrderStatusRequest{Status: "processing"}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, "/orders/"+ordID.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, userID, []string{"patient"}) // Only pharmacy_staff can update status

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockSvc.AssertNotCalled(t, "UpdateStatus")
}

func TestOrderHandler_UpdateStatus_SuccessByStaff(t *testing.T) {
	mockSvc := new(MockOrderService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	ordID := uuid.New()
	reqBody := dto.UpdateOrderStatusRequest{Status: "processing"}

	expectedResp := &dto.OrderResponse{
		ID:          ordID.String(),
		Status:      "processing",
		TotalAmount: 150000.0,
	}

	mockSvc.On("UpdateStatus", mock.Anything, userID, ordID, reqBody).Return(expectedResp, nil).Once()

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, "/orders/"+ordID.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, userID, []string{"pharmacy_staff"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}
