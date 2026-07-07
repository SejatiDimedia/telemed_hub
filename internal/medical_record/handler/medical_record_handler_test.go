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
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type MockMedicalRecordService struct {
	mock.Mock
}

func (m *MockMedicalRecordService) Create(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, req dto.CreateMedicalRecordRequest) (*dto.MedicalRecordResponse, error) {
	args := m.Called(ctx, actorID, roles, ipAddress, userAgent, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MedicalRecordResponse), args.Error(1)
}

func (m *MockMedicalRecordService) GetByID(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, id uuid.UUID) (*dto.MedicalRecordResponse, error) {
	args := m.Called(ctx, actorID, roles, ipAddress, userAgent, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MedicalRecordResponse), args.Error(1)
}

func (m *MockMedicalRecordService) List(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, filter dto.ListMedicalRecordsFilter) ([]*dto.MedicalRecordResponse, error) {
	args := m.Called(ctx, actorID, roles, ipAddress, userAgent, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*dto.MedicalRecordResponse), args.Error(1)
}

func (m *MockMedicalRecordService) Update(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, id uuid.UUID, req dto.UpdateMedicalRecordRequest) (*dto.MedicalRecordResponse, error) {
	args := m.Called(ctx, actorID, roles, ipAddress, userAgent, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MedicalRecordResponse), args.Error(1)
}

func (m *MockMedicalRecordService) Delete(ctx context.Context, actorID uuid.UUID, roles []string, ipAddress, userAgent string, id uuid.UUID) error {
	args := m.Called(ctx, actorID, roles, ipAddress, userAgent, id)
	return args.Error(0)
}

var _ service.MedicalRecordService = (*MockMedicalRecordService)(nil)

func newTestHandler(svc service.MedicalRecordService) (*MedicalRecordHandler, chi.Router) {
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewMedicalRecordHandler(svc, cfg, nil, log)

	r := chi.NewRouter()
	r.Post("/medical-records", h.Create)
	r.Get("/medical-records/{id}", h.GetByID)
	r.Get("/medical-records", h.List)
	r.Put("/medical-records/{id}", h.Update)
	r.Delete("/medical-records/{id}", h.Delete)

	return h, r
}

func withAuth(req *http.Request, userID uuid.UUID, roles []string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, middleware.RolesContextKey, roles)
	return req.WithContext(ctx)
}

func TestMedicalRecordHandler_Create_Success(t *testing.T) {
	mockSvc := new(MockMedicalRecordService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	patientID := uuid.New().String()
	reqBody := dto.CreateMedicalRecordRequest{
		PatientID:  patientID,
		RecordType: "diagnosis",
		Content:    "Patient has allergies",
	}

	expectedResp := &dto.MedicalRecordResponse{
		ID:         uuid.New().String(),
		PatientID:  patientID,
		RecordType: "diagnosis",
		Content:    "Patient has allergies",
	}

	mockSvc.On("Create", mock.Anything, userID, []string{"doctor"}, mock.Anything, mock.Anything, reqBody).Return(expectedResp, nil).Once()

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/medical-records", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, userID, []string{"doctor"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	mockSvc.AssertExpectations(t)
}

func TestMedicalRecordHandler_Create_ForbiddenForPatient(t *testing.T) {
	mockSvc := new(MockMedicalRecordService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	reqBody := dto.CreateMedicalRecordRequest{
		PatientID:  uuid.New().String(),
		RecordType: "diagnosis",
		Content:    "Symptom notes",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/medical-records", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, userID, []string{"patient"}) // Patients cannot create records

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockSvc.AssertNotCalled(t, "Create")
}

func TestMedicalRecordHandler_GetByID_Success(t *testing.T) {
	mockSvc := new(MockMedicalRecordService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	recordID := uuid.New()
	expectedResp := &dto.MedicalRecordResponse{
		ID:         recordID.String(),
		PatientID:  uuid.New().String(),
		RecordType: "lab_result",
		Content:    "Blood test clear",
	}

	mockSvc.On("GetByID", mock.Anything, userID, []string{"doctor"}, mock.Anything, mock.Anything, recordID).Return(expectedResp, nil).Once()

	req, _ := http.NewRequest(http.MethodGet, "/medical-records/"+recordID.String(), nil)
	req = withAuth(req, userID, []string{"doctor"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}
