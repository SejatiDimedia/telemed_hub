package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type MockAppointmentService struct {
	mock.Mock
}

func (m *MockAppointmentService) Book(ctx context.Context, patientUserID uuid.UUID, req dto.CreateAppointmentRequest) (*dto.AppointmentResponse, error) {
	args := m.Called(ctx, patientUserID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AppointmentResponse), args.Error(1)
}

func (m *MockAppointmentService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.AppointmentResponse, error) {
	args := m.Called(ctx, id, userID, roles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AppointmentResponse), args.Error(1)
}

func (m *MockAppointmentService) List(ctx context.Context, userID uuid.UUID, roles []string, statusFilter string) ([]*dto.AppointmentResponse, error) {
	args := m.Called(ctx, userID, roles, statusFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*dto.AppointmentResponse), args.Error(1)
}

func (m *MockAppointmentService) Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CancelAppointmentRequest) error {
	args := m.Called(ctx, id, userID, req)
	return args.Error(0)
}

func (m *MockAppointmentService) Reschedule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.RescheduleAppointmentRequest) (*dto.AppointmentResponse, error) {
	args := m.Called(ctx, id, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AppointmentResponse), args.Error(1)
}

func (m *MockAppointmentService) SetConsultationService(consSvc service.ConsultationServiceClient) {
	m.Called(consSvc)
}

func TestAppointmentHandler_Book(t *testing.T) {
	mockSvc := new(MockAppointmentService)
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewAppointmentHandler(mockSvc, cfg, nil, log)

	r := chi.NewRouter()
	r.Post("/appointments", h.Book)

	patientUserID := uuid.New()
	doctorID := uuid.New()
	availID := uuid.New()

	t.Run("Success booking", func(t *testing.T) {
		reqBody := dto.CreateAppointmentRequest{
			DoctorID:       doctorID.String(),
			AvailabilityID: availID.String(),
		}

		expectedResp := &dto.AppointmentResponse{
			ID:             uuid.New().String(),
			PatientID:      uuid.New().String(),
			DoctorID:       doctorID.String(),
			AvailabilityID: availID.String(),
			Status:         "confirmed",
			ScheduledAt:    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}

		mockSvc.On("Book", mock.Anything, patientUserID, reqBody).Return(expectedResp, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/appointments", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, patientUserID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"patient"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var envelope struct {
			Success bool                    `json:"success"`
			Data    dto.AppointmentResponse `json:"data"`
		}
		err := json.NewDecoder(rec.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Equal(t, expectedResp.ID, envelope.Data.ID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Insufficient balance returns 422", func(t *testing.T) {
		reqBody := dto.CreateAppointmentRequest{
			DoctorID:       doctorID.String(),
			AvailabilityID: availID.String(),
		}

		mockSvc.On("Book", mock.Anything, patientUserID, reqBody).Return(nil, service.ErrInsufficientBalance).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/appointments", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, patientUserID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"patient"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		var envelope struct {
			Success   bool   `json:"success"`
			Error     string `json:"error"`
			ErrorCode string `json:"error_code"`
		}
		_ = json.NewDecoder(rec.Body).Decode(&envelope)
		assert.False(t, envelope.Success)
		assert.Equal(t, "INSUFFICIENT_BALANCE", envelope.ErrorCode)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Double booking conflict returns 409", func(t *testing.T) {
		reqBody := dto.CreateAppointmentRequest{
			DoctorID:       doctorID.String(),
			AvailabilityID: availID.String(),
		}

		mockSvc.On("Book", mock.Anything, patientUserID, reqBody).Return(nil, repository.ErrSlotAlreadyBooked).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/appointments", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, patientUserID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"patient"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		mockSvc.AssertExpectations(t)
	})
}
