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
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type MockConsultationService struct {
	mock.Mock
}

func (m *MockConsultationService) CreateConsultation(ctx context.Context, appointmentID uuid.UUID) error {
	args := m.Called(ctx, appointmentID)
	return args.Error(0)
}

func (m *MockConsultationService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.ConsultationResponse, error) {
	args := m.Called(ctx, id, userID, roles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ConsultationResponse), args.Error(1)
}

func (m *MockConsultationService) Start(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID) (*dto.ConsultationResponse, error) {
	args := m.Called(ctx, id, doctorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ConsultationResponse), args.Error(1)
}

func (m *MockConsultationService) Complete(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID) (*dto.ConsultationResponse, error) {
	args := m.Called(ctx, id, doctorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ConsultationResponse), args.Error(1)
}

func (m *MockConsultationService) UpdateNotes(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID, notes string) (*dto.ConsultationResponse, error) {
	args := m.Called(ctx, id, doctorUserID, notes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ConsultationResponse), args.Error(1)
}

func (m *MockConsultationService) CancelConsultation(ctx context.Context, appointmentID uuid.UUID) error {
	args := m.Called(ctx, appointmentID)
	return args.Error(0)
}

func TestConsultationHandler_Start(t *testing.T) {
	mockSvc := new(MockConsultationService)
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewConsultationHandler(mockSvc, cfg, nil, log)

	r := chi.NewRouter()
	r.Post("/consultations/{id}/start", h.Start)

	consID := uuid.New()
	doctorUserID := uuid.New()

	t.Run("Success start consultation", func(t *testing.T) {
		startedStr := "2026-07-04T10:00:00Z"
		expectedResp := &dto.ConsultationResponse{
			ID:            consID.String(),
			AppointmentID: uuid.New().String(),
			Status:        "in_progress",
			StartedAt:     &startedStr,
		}

		mockSvc.On("Start", mock.Anything, consID, doctorUserID).Return(expectedResp, nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/consultations/"+consID.String()+"/start", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, doctorUserID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"doctor"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var envelope struct {
			Success bool                     `json:"success"`
			Data    dto.ConsultationResponse `json:"data"`
		}
		err := json.NewDecoder(rec.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Equal(t, "in_progress", envelope.Data.Status)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Invalid state transition returns 422", func(t *testing.T) {
		mockSvc.On("Start", mock.Anything, consID, doctorUserID).Return(nil, service.ErrInvalidTransition).Once()

		req, _ := http.NewRequest(http.MethodPost, "/consultations/"+consID.String()+"/start", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, doctorUserID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"doctor"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		mockSvc.AssertExpectations(t)
	})
}
