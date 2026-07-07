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
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
	"github.com/jackc/pgx/v5"
)

type MockInventoryService struct {
	mock.Mock
}

func (m *MockInventoryService) Create(ctx context.Context, adminUserID uuid.UUID, req dto.CreateMedicineRequest) (*dto.MedicineResponse, error) {
	args := m.Called(ctx, adminUserID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MedicineResponse), args.Error(1)
}

func (m *MockInventoryService) Update(ctx context.Context, adminUserID uuid.UUID, id uuid.UUID, req dto.UpdateMedicineRequest) (*dto.MedicineResponse, error) {
	args := m.Called(ctx, adminUserID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MedicineResponse), args.Error(1)
}

func (m *MockInventoryService) GetByID(ctx context.Context, id uuid.UUID) (*dto.MedicineResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MedicineResponse), args.Error(1)
}

func (m *MockInventoryService) List(ctx context.Context, nameFilter *string, reqPrescFilter *bool, page, limit int) ([]*dto.MedicineResponse, int, error) {
	args := m.Called(ctx, nameFilter, reqPrescFilter, page, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*dto.MedicineResponse), args.Int(1), args.Error(2)
}

func (m *MockInventoryService) DecrementStock(ctx context.Context, tx pgx.Tx, id uuid.UUID, qty int) error {
	args := m.Called(ctx, tx, id, qty)
	return args.Error(0)
}

func (m *MockInventoryService) IncrementStock(ctx context.Context, tx pgx.Tx, id uuid.UUID, qty int) error {
	args := m.Called(ctx, tx, id, qty)
	return args.Error(0)
}

func (m *MockInventoryService) Delete(ctx context.Context, adminUserID uuid.UUID, id uuid.UUID) error {
	args := m.Called(ctx, adminUserID, id)
	return args.Error(0)
}

var _ service.InventoryService = (*MockInventoryService)(nil)

func newTestHandler(svc service.InventoryService) (*InventoryHandler, chi.Router) {
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewInventoryHandler(svc, cfg, nil, log)

	r := chi.NewRouter()
	r.Post("/medicines", h.Create)
	r.Get("/medicines/{id}", h.GetByID)
	r.Get("/medicines", h.List)
	r.Put("/medicines/{id}", h.Update)
	r.Delete("/medicines/{id}", h.Delete)

	return h, r
}

func withAuth(req *http.Request, userID uuid.UUID, roles []string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, middleware.RolesContextKey, roles)
	return req.WithContext(ctx)
}

func TestInventoryHandler_Create_Success(t *testing.T) {
	mockSvc := new(MockInventoryService)
	_, r := newTestHandler(mockSvc)

	adminUserID := uuid.New()
	reqBody := dto.CreateMedicineRequest{
		Name:                 "Paracetamol 500mg",
		UnitPrice:            6000.0,
		StockQuantity:        100,
		RequiresPrescription: false,
	}

	expectedResp := &dto.MedicineResponse{
		ID:                   uuid.New().String(),
		Name:                 "Paracetamol 500mg",
		UnitPrice:            6000.0,
		StockQuantity:        100,
		RequiresPrescription: false,
		CreatedAt:            "2026-07-06T00:00:00Z",
		UpdatedAt:            "2026-07-06T00:00:00Z",
	}

	mockSvc.On("Create", mock.Anything, adminUserID, reqBody).Return(expectedResp, nil).Once()

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/medicines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, adminUserID, []string{"admin"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var envelope struct {
		Success bool                 `json:"success"`
		Data    dto.MedicineResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&envelope))
	assert.True(t, envelope.Success)
	assert.Equal(t, "Paracetamol 500mg", envelope.Data.Name)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_Create_ForbiddenForPatient(t *testing.T) {
	mockSvc := new(MockInventoryService)
	_, r := newTestHandler(mockSvc)

	patientUserID := uuid.New()
	reqBody := dto.CreateMedicineRequest{
		Name:                 "Paracetamol 500mg",
		UnitPrice:            6000.0,
		StockQuantity:        100,
		RequiresPrescription: false,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/medicines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, patientUserID, []string{"patient"}) // Not admin or pharmacy_staff

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockSvc.AssertNotCalled(t, "Create")
}

func TestInventoryHandler_Create_ValidationError(t *testing.T) {
	mockSvc := new(MockInventoryService)
	_, r := newTestHandler(mockSvc)

	adminUserID := uuid.New()
	reqBody := dto.CreateMedicineRequest{
		Name:                 "", // Empty name triggers validation error
		UnitPrice:            6000.0,
		StockQuantity:        100,
		RequiresPrescription: false,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/medicines", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, adminUserID, []string{"admin"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	mockSvc.AssertNotCalled(t, "Create")
}

func TestInventoryHandler_GetByID_Success(t *testing.T) {
	mockSvc := new(MockInventoryService)
	_, r := newTestHandler(mockSvc)

	medID := uuid.New()
	userID := uuid.New()

	expectedResp := &dto.MedicineResponse{
		ID:                   medID.String(),
		Name:                 "Amoxicillin 500mg",
		UnitPrice:            15000.0,
		StockQuantity:        20,
		RequiresPrescription: true,
	}

	mockSvc.On("GetByID", mock.Anything, medID).Return(expectedResp, nil).Once()

	req, _ := http.NewRequest(http.MethodGet, "/medicines/"+medID.String(), nil)
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var envelope struct {
		Success bool                 `json:"success"`
		Data    dto.MedicineResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&envelope))
	assert.True(t, envelope.Success)
	assert.Equal(t, "Amoxicillin 500mg", envelope.Data.Name)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_GetByID_NotFound(t *testing.T) {
	mockSvc := new(MockInventoryService)
	_, r := newTestHandler(mockSvc)

	medID := uuid.New()
	userID := uuid.New()

	mockSvc.On("GetByID", mock.Anything, medID).Return(nil, service.ErrMedicineNotFound).Once()

	req, _ := http.NewRequest(http.MethodGet, "/medicines/"+medID.String(), nil)
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_List_Success(t *testing.T) {
	mockSvc := new(MockInventoryService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	medicines := []*dto.MedicineResponse{
		{
			ID:                   uuid.New().String(),
			Name:                 "Cetirizine 10mg",
			UnitPrice:            5000.0,
			StockQuantity:        100,
			RequiresPrescription: false,
		},
	}

	var nameFilter *string
	var reqPrescFilter *bool

	mockSvc.On("List", mock.Anything, nameFilter, reqPrescFilter, 1, 20).Return(medicines, 1, nil).Once()

	req, _ := http.NewRequest(http.MethodGet, "/medicines", nil)
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var envelope struct {
		Success    bool                   `json:"success"`
		Data       []dto.MedicineResponse `json:"data"`
		Pagination struct {
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			TotalItems int `json:"total_items"`
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&envelope))
	assert.True(t, envelope.Success)
	assert.Len(t, envelope.Data, 1)
	assert.Equal(t, "Cetirizine 10mg", envelope.Data[0].Name)
	assert.Equal(t, 1, envelope.Pagination.TotalItems)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_Update_Success(t *testing.T) {
	mockSvc := new(MockInventoryService)
	_, r := newTestHandler(mockSvc)

	adminUserID := uuid.New()
	medID := uuid.New()
	reqBody := dto.UpdateMedicineRequest{
		Name:                 "Cetirizine Updated",
		UnitPrice:            6000.0,
		StockQuantity:        200,
		RequiresPrescription: false,
	}

	expectedResp := &dto.MedicineResponse{
		ID:                   medID.String(),
		Name:                 "Cetirizine Updated",
		UnitPrice:            6000.0,
		StockQuantity:        200,
		RequiresPrescription: false,
	}

	mockSvc.On("Update", mock.Anything, adminUserID, medID, reqBody).Return(expectedResp, nil).Once()

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, "/medicines/"+medID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, adminUserID, []string{"admin"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}

func TestInventoryHandler_Delete_Success(t *testing.T) {
	mockSvc := new(MockInventoryService)
	_, r := newTestHandler(mockSvc)

	adminUserID := uuid.New()
	medID := uuid.New()

	mockSvc.On("Delete", mock.Anything, adminUserID, medID).Return(nil).Once()

	req, _ := http.NewRequest(http.MethodDelete, "/medicines/"+medID.String(), nil)
	req = withAuth(req, adminUserID, []string{"admin"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}
