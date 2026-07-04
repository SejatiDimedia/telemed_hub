package handler

import (
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
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type MockPatientService struct {
	mock.Mock
}

func (m *MockPatientService) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*dto.PatientResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PatientResponse), args.Error(1)
}

func (m *MockPatientService) GetProfileByID(ctx context.Context, id uuid.UUID) (*dto.PatientResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PatientResponse), args.Error(1)
}

func (m *MockPatientService) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdatePatientRequest) (*dto.PatientResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PatientResponse), args.Error(1)
}

func TestPatientHandler_GetMe(t *testing.T) {
	mockSvc := new(MockPatientService)
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewPatientHandler(mockSvc, cfg, nil, log)

	r := chi.NewRouter()
	r.Get("/patients/me", h.GetMe)

	t.Run("Success Get Me", func(t *testing.T) {
		userID := uuid.New()
		expectedResp := &dto.PatientResponse{
			ID:       uuid.New().String(),
			UserID:   userID.String(),
			Email:    "rina@test.com",
			FullName: "Rina Wijaya",
		}

		mockSvc.On("GetProfileByUserID", mock.Anything, userID).Return(expectedResp, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/patients/me", nil)
		// Inject userID into ctx manually to simulate AuthMiddleware injection
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"patient"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var envelope struct {
			Success bool                `json:"success"`
			Data    dto.PatientResponse `json:"data"`
		}
		err := json.NewDecoder(rec.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Equal(t, expectedResp.FullName, envelope.Data.FullName)
		mockSvc.AssertExpectations(t)
	})
}

func TestPatientHandler_GetByID(t *testing.T) {
	mockSvc := new(MockPatientService)
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewPatientHandler(mockSvc, cfg, nil, log)

	r := chi.NewRouter()
	r.Get("/patients/{id}", h.GetByID)

	t.Run("Success view own profile", func(t *testing.T) {
		userID := uuid.New()
		patientID := uuid.New()
		expectedResp := &dto.PatientResponse{
			ID:     patientID.String(),
			UserID: userID.String(),
			Email:  "rina@test.com",
		}

		mockSvc.On("GetProfileByID", mock.Anything, patientID).Return(expectedResp, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/patients/"+patientID.String(), nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"patient"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Forbidden view another patient profile", func(t *testing.T) {
		userID := uuid.New() // Logged-in user
		anotherUserID := uuid.New()
		patientID := uuid.New()
		expectedResp := &dto.PatientResponse{
			ID:     patientID.String(),
			UserID: anotherUserID.String(), // Different owner
			Email:  "another@test.com",
		}

		mockSvc.On("GetProfileByID", mock.Anything, patientID).Return(expectedResp, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/patients/"+patientID.String(), nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"patient"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Doctor allowed to view profile", func(t *testing.T) {
		doctorUserID := uuid.New()
		patientUserID := uuid.New()
		patientID := uuid.New()
		expectedResp := &dto.PatientResponse{
			ID:     patientID.String(),
			UserID: patientUserID.String(),
		}

		mockSvc.On("GetProfileByID", mock.Anything, patientID).Return(expectedResp, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/patients/"+patientID.String(), nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, doctorUserID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"doctor"}) // Doctor role
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		mockSvc.AssertExpectations(t)
	})
}
