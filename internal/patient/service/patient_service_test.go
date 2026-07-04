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

	"github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/validator"
)

type MockPatientRepository struct {
	mock.Mock
}

func (m *MockPatientRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Patient, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Patient), args.Error(1)
}

func (m *MockPatientRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Patient, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Patient), args.Error(1)
}

func (m *MockPatientRepository) Update(ctx context.Context, patient *model.Patient) error {
	args := m.Called(ctx, patient)
	return args.Error(0)
}

func TestPatientService_GetProfileByUserID(t *testing.T) {
	mockRepo := new(MockPatientRepository)
	svc := NewPatientService(mockRepo)

	userID := uuid.New()
	pID := uuid.New()
	dob, _ := time.Parse("2006-01-02", "1995-04-12")
	gender := "female"
	bloodType := "O+"
	phone := "+6281234567890"

	patient := &model.Patient{
		ID:          pID,
		UserID:      userID,
		Email:       "test@patient.com",
		FullName:    "Test Patient",
		PhoneNumber: &phone,
		DateOfBirth: &dob,
		Gender:      &gender,
		BloodType:   &bloodType,
	}

	t.Run("Success profile retrieval", func(t *testing.T) {
		mockRepo.On("GetByUserID", mock.Anything, userID).Return(patient, nil).Once()

		resp, err := svc.GetProfileByUserID(context.Background(), userID)
		require.NoError(t, err)
		assert.Equal(t, pID.String(), resp.ID)
		assert.Equal(t, "1995-04-12", *resp.DateOfBirth)
		assert.Equal(t, "female", *resp.Gender)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Profile not found", func(t *testing.T) {
		mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, repository.ErrPatientNotFound).Once()

		resp, err := svc.GetProfileByUserID(context.Background(), userID)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, repository.ErrPatientNotFound)
		mockRepo.AssertExpectations(t)
	})
}

func TestPatientService_UpdateProfile(t *testing.T) {
	mockRepo := new(MockPatientRepository)
	svc := NewPatientService(mockRepo)

	userID := uuid.New()
	pID := uuid.New()
	phoneBefore := "+628111111111"
	patientBefore := &model.Patient{
		ID:          pID,
		UserID:      userID,
		Email:       "test@patient.com",
		FullName:    "Test Patient",
		PhoneNumber: &phoneBefore,
	}

	t.Run("Success update", func(t *testing.T) {
		req := dto.UpdatePatientRequest{
			DateOfBirth: "1995-04-12",
			Gender:      "female",
			BloodType:   ptr("O+"),
			PhoneNumber: "+6281234567890",
		}

		mockRepo.On("GetByUserID", mock.Anything, userID).Return(patientBefore, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Once()

		resp, err := svc.UpdateProfile(context.Background(), userID, req)
		require.NoError(t, err)
		assert.Equal(t, "1995-04-12", *resp.DateOfBirth)
		assert.Equal(t, "female", *resp.Gender)
		assert.Equal(t, "O+", *resp.BloodType)
		assert.Equal(t, "+6281234567890", *resp.PhoneNumber)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Validation failure", func(t *testing.T) {
		req := dto.UpdatePatientRequest{
			DateOfBirth: "future-date-2099-01-01", // will fail format or past date check
			Gender:      "alien",
			BloodType:   ptr("Z"),
			PhoneNumber: "invalid-phone",
		}

		resp, err := svc.UpdateProfile(context.Background(), userID, req)
		assert.Nil(t, resp)
		var valErrs validator.ValidationErrors
		assert.True(t, errors.As(err, &valErrs))
		assert.Len(t, valErrs, 4) // 4 issues expected
	})
}

func ptr(s string) *string {
	return &s
}
