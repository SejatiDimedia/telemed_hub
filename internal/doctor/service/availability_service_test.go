package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/validator"
)

func TestDoctorService_AddAvailability(t *testing.T) {
	mockRepo := new(MockDoctorRepository)
	svc := NewDoctorService(mockRepo, nil)

	doctorUserID := uuid.New()
	doctorID := uuid.New()
	doctor := &model.Doctor{
		ID:     doctorID,
		UserID: doctorUserID,
	}

	t.Run("Success add slot", func(t *testing.T) {
		req := dto.CreateAvailabilityRequest{
			StartTime: time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
			EndTime:   time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339),
		}

		mockRepo.On("GetByUserID", mock.Anything, doctorUserID).Return(doctor, nil).Once()
		mockRepo.On("CheckOverlappingSlot", mock.Anything, doctorID, mock.Anything, mock.Anything).Return(false, nil).Once()
		mockRepo.On("CreateAvailability", mock.Anything, mock.Anything).Return(nil).Once()

		resp, err := svc.AddAvailability(context.Background(), doctorUserID, req)
		require.NoError(t, err)
		assert.Equal(t, doctorID.String(), resp.DoctorID)
		assert.False(t, resp.IsBooked)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Overlap conflict error", func(t *testing.T) {
		req := dto.CreateAvailabilityRequest{
			StartTime: time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
			EndTime:   time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339),
		}

		mockRepo.On("GetByUserID", mock.Anything, doctorUserID).Return(doctor, nil).Once()
		mockRepo.On("CheckOverlappingSlot", mock.Anything, doctorID, mock.Anything, mock.Anything).Return(true, nil).Once()

		resp, err := svc.AddAvailability(context.Background(), doctorUserID, req)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrOverlappingSlot)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Validation formatting error", func(t *testing.T) {
		req := dto.CreateAvailabilityRequest{
			StartTime: "invalid-time",
			EndTime:   "invalid-time",
		}

		mockRepo.On("GetByUserID", mock.Anything, doctorUserID).Return(doctor, nil).Once()

		resp, err := svc.AddAvailability(context.Background(), doctorUserID, req)
		assert.Nil(t, resp)
		var valErrs validator.ValidationErrors
		assert.True(t, errors.As(err, &valErrs))
		assert.Len(t, valErrs, 2)
		mockRepo.AssertExpectations(t)
	})
}

func TestDoctorService_RemoveAvailability(t *testing.T) {
	mockRepo := new(MockDoctorRepository)
	svc := NewDoctorService(mockRepo, nil)

	doctorUserID := uuid.New()
	doctorID := uuid.New()
	doctor := &model.Doctor{
		ID:     doctorID,
		UserID: doctorUserID,
	}
	slotID := uuid.New()

	t.Run("Success remove slot", func(t *testing.T) {
		mockRepo.On("GetByUserID", mock.Anything, doctorUserID).Return(doctor, nil).Once()
		mockRepo.On("DeleteAvailability", mock.Anything, doctorID, slotID).Return(nil).Once()

		err := svc.RemoveAvailability(context.Background(), doctorUserID, slotID)
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Delete booked slot returns error", func(t *testing.T) {
		mockRepo.On("GetByUserID", mock.Anything, doctorUserID).Return(doctor, nil).Once()
		mockRepo.On("DeleteAvailability", mock.Anything, doctorID, slotID).Return(repository.ErrSlotBooked).Once()

		err := svc.RemoveAvailability(context.Background(), doctorUserID, slotID)
		assert.ErrorIs(t, err, repository.ErrSlotBooked)
		mockRepo.AssertExpectations(t)
	})
}
