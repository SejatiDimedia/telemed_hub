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
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

// MockPrescriptionService is a testify mock for service.PrescriptionService.
type MockPrescriptionService struct{ mock.Mock }

func (m *MockPrescriptionService) Issue(ctx context.Context, doctorUserID uuid.UUID, req dto.CreatePrescriptionRequest) (*dto.PrescriptionResponse, error) {
	args := m.Called(ctx, doctorUserID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PrescriptionResponse), args.Error(1)
}

func (m *MockPrescriptionService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.PrescriptionResponse, error) {
	args := m.Called(ctx, id, userID, roles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PrescriptionResponse), args.Error(1)
}

func (m *MockPrescriptionService) List(ctx context.Context, userID uuid.UUID, roles []string) ([]*dto.PrescriptionResponse, error) {
	args := m.Called(ctx, userID, roles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*dto.PrescriptionResponse), args.Error(1)
}

// Compile-time interface assertion
var _ service.PrescriptionService = (*MockPrescriptionService)(nil)

func newTestHandler(svc service.PrescriptionService) (*PrescriptionHandler, chi.Router) {
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewPrescriptionHandler(svc, cfg, nil, log)

	r := chi.NewRouter()
	r.Post("/prescriptions", h.Issue)
	r.Get("/prescriptions/{id}", h.GetByID)
	r.Get("/prescriptions", h.List)

	return h, r
}

// withAuth injects UserID and roles into the request context (simulates passed middleware).
func withAuth(req *http.Request, userID uuid.UUID, roles []string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, middleware.RolesContextKey, roles)
	return req.WithContext(ctx)
}

// --- Tests: Issue ---

func TestPrescriptionHandler_Issue_Success(t *testing.T) {
	mockSvc := new(MockPrescriptionService)
	_, r := newTestHandler(mockSvc)

	doctorUserID := uuid.New()
	prescriptionID := uuid.New()
	medicineID := uuid.New()
	consultationID := uuid.New()

	reqBody := dto.CreatePrescriptionRequest{
		ConsultationID: consultationID.String(),
		Items: []dto.PrescriptionItemRequest{
			{MedicineID: medicineID.String(), Dosage: "500mg twice daily", Quantity: 10},
		},
	}

	expectedResp := &dto.PrescriptionResponse{
		ID:     prescriptionID.String(),
		Status: "active",
		Items: []dto.PrescriptionItemResponse{
			{ID: uuid.New().String(), MedicineID: medicineID.String(), MedicineName: "Amoxicillin 500mg", Dosage: "500mg twice daily", Quantity: 10},
		},
	}

	mockSvc.On("Issue", mock.Anything, doctorUserID, reqBody).Return(expectedResp, nil).Once()

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/prescriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, doctorUserID, []string{"doctor"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var envelope struct {
		Success bool                    `json:"success"`
		Data    dto.PrescriptionResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&envelope))
	assert.True(t, envelope.Success)
	assert.Equal(t, "active", envelope.Data.Status)
	assert.Len(t, envelope.Data.Items, 1)
	mockSvc.AssertExpectations(t)
}

func TestPrescriptionHandler_Issue_ForbiddenForNonDoctor(t *testing.T) {
	mockSvc := new(MockPrescriptionService)
	_, r := newTestHandler(mockSvc)

	reqBody := dto.CreatePrescriptionRequest{
		ConsultationID: uuid.New().String(),
		Items:          []dto.PrescriptionItemRequest{{MedicineID: uuid.New().String(), Dosage: "10mg", Quantity: 1}},
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/prescriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, uuid.New(), []string{"patient"}) // patient role, not doctor

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockSvc.AssertNotCalled(t, "Issue")
}

func TestPrescriptionHandler_Issue_ValidationError_EmptyItems(t *testing.T) {
	mockSvc := new(MockPrescriptionService)
	_, r := newTestHandler(mockSvc)

	reqBody := dto.CreatePrescriptionRequest{
		ConsultationID: uuid.New().String(),
		Items:          []dto.PrescriptionItemRequest{}, // empty items
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/prescriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, uuid.New(), []string{"doctor"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	mockSvc.AssertNotCalled(t, "Issue")
}

func TestPrescriptionHandler_Issue_UnprocessableWhenConsultationScheduled(t *testing.T) {
	mockSvc := new(MockPrescriptionService)
	_, r := newTestHandler(mockSvc)

	doctorUserID := uuid.New()
	consultationID := uuid.New()
	medicineID := uuid.New()

	reqBody := dto.CreatePrescriptionRequest{
		ConsultationID: consultationID.String(),
		Items:          []dto.PrescriptionItemRequest{{MedicineID: medicineID.String(), Dosage: "10mg", Quantity: 5}},
	}
	mockSvc.On("Issue", mock.Anything, doctorUserID, reqBody).Return(nil, service.ErrInvalidConsultationStatus).Once()

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/prescriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, doctorUserID, []string{"doctor"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	mockSvc.AssertExpectations(t)
}

// --- Tests: GetByID ---

func TestPrescriptionHandler_GetByID_Success(t *testing.T) {
	mockSvc := new(MockPrescriptionService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	prescriptionID := uuid.New()
	expectedResp := &dto.PrescriptionResponse{
		ID:     prescriptionID.String(),
		Status: "active",
		Items:  []dto.PrescriptionItemResponse{},
	}
	mockSvc.On("GetByID", mock.Anything, prescriptionID, userID, []string{"patient"}).Return(expectedResp, nil).Once()

	req, _ := http.NewRequest(http.MethodGet, "/prescriptions/"+prescriptionID.String(), nil)
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}

func TestPrescriptionHandler_GetByID_Unauthorized(t *testing.T) {
	mockSvc := new(MockPrescriptionService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	prescriptionID := uuid.New()
	mockSvc.On("GetByID", mock.Anything, prescriptionID, userID, []string{"patient"}).Return(nil, service.ErrUnauthorized).Once()

	req, _ := http.NewRequest(http.MethodGet, "/prescriptions/"+prescriptionID.String(), nil)
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockSvc.AssertExpectations(t)
}

func TestPrescriptionHandler_GetByID_NotFound(t *testing.T) {
	mockSvc := new(MockPrescriptionService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	prescriptionID := uuid.New()
	mockSvc.On("GetByID", mock.Anything, prescriptionID, userID, []string{"doctor"}).Return(nil, service.ErrPrescriptionNotFound).Once()

	req, _ := http.NewRequest(http.MethodGet, "/prescriptions/"+prescriptionID.String(), nil)
	req = withAuth(req, userID, []string{"doctor"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	mockSvc.AssertExpectations(t)
}

// --- Tests: List ---

func TestPrescriptionHandler_List_Success(t *testing.T) {
	mockSvc := new(MockPrescriptionService)
	_, r := newTestHandler(mockSvc)

	doctorUserID := uuid.New()
	mockSvc.On("List", mock.Anything, doctorUserID, []string{"doctor"}).Return([]*dto.PrescriptionResponse{
		{ID: uuid.New().String(), Status: "active", Items: []dto.PrescriptionItemResponse{}},
	}, nil).Once()

	req, _ := http.NewRequest(http.MethodGet, "/prescriptions", nil)
	req = withAuth(req, doctorUserID, []string{"doctor"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}
