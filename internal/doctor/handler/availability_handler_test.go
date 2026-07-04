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

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

func TestDoctorHandler_AddAvailabilitySlot(t *testing.T) {
	mockSvc := new(MockDoctorService)
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewDoctorHandler(mockSvc, cfg, nil, log)

	r := chi.NewRouter()
	r.Post("/doctors/me/availability", h.AddAvailabilitySlot)

	doctorUserID := uuid.New()
	startTime := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	endTime := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)

	t.Run("Success add availability", func(t *testing.T) {
		reqBody := dto.CreateAvailabilityRequest{
			StartTime: startTime,
			EndTime:   endTime,
		}

		expectedResp := &dto.AvailabilityResponse{
			ID:        uuid.New().String(),
			DoctorID:  uuid.New().String(),
			StartTime: startTime,
			EndTime:   endTime,
			IsBooked:  false,
		}

		mockSvc.On("AddAvailability", mock.Anything, doctorUserID, reqBody).Return(expectedResp, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/doctors/me/availability", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, doctorUserID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"doctor"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var envelope struct {
			Success bool                   `json:"success"`
			Data    dto.AvailabilityResponse `json:"data"`
		}
		err := json.NewDecoder(rec.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Equal(t, expectedResp.ID, envelope.Data.ID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Conflict overlap returns 409", func(t *testing.T) {
		reqBody := dto.CreateAvailabilityRequest{
			StartTime: startTime,
			EndTime:   endTime,
		}

		mockSvc.On("AddAvailability", mock.Anything, doctorUserID, reqBody).Return(nil, service.ErrOverlappingSlot).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/doctors/me/availability", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, doctorUserID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"doctor"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestDoctorHandler_RemoveAvailabilitySlot(t *testing.T) {
	mockSvc := new(MockDoctorService)
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewDoctorHandler(mockSvc, cfg, nil, log)

	r := chi.NewRouter()
	r.Delete("/doctors/me/availability/{slotId}", h.RemoveAvailabilitySlot)

	doctorUserID := uuid.New()
	slotID := uuid.New()

	t.Run("Success remove availability", func(t *testing.T) {
		mockSvc.On("RemoveAvailability", mock.Anything, doctorUserID, slotID).Return(nil).Once()

		req, _ := http.NewRequest(http.MethodDelete, "/doctors/me/availability/"+slotID.String(), nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, doctorUserID)
		ctx = context.WithValue(ctx, middleware.RolesContextKey, []string{"doctor"})
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		mockSvc.AssertExpectations(t)
	})
}
