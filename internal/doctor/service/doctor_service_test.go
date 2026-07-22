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
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/validator"
	"github.com/timurdianradhasejati/telemed_hub/internal/shared"
)

type MockDoctorRepository struct {
	mock.Mock
}

func (m *MockDoctorRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Doctor, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Doctor), args.Error(1)
}

func (m *MockDoctorRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Doctor, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Doctor), args.Error(1)
}

func (m *MockDoctorRepository) Update(ctx context.Context, doctor *model.Doctor) error {
	args := m.Called(ctx, doctor)
	return args.Error(0)
}

func (m *MockDoctorRepository) Verify(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDoctorRepository) List(ctx context.Context, specialty *string, onlyVerified bool, sortBy string, order string, offset int, limit int) ([]*model.Doctor, int, error) {
	args := m.Called(ctx, specialty, onlyVerified, sortBy, order, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*model.Doctor), args.Int(1), args.Error(2)
}

func (m *MockDoctorRepository) CreateAvailability(ctx context.Context, slot *model.Availability) error {
	args := m.Called(ctx, slot)
	return args.Error(0)
}

func (m *MockDoctorRepository) DeleteAvailability(ctx context.Context, doctorID uuid.UUID, slotID uuid.UUID) error {
	args := m.Called(ctx, doctorID, slotID)
	return args.Error(0)
}

func (m *MockDoctorRepository) GetAvailabilityByID(ctx context.Context, slotID uuid.UUID) (*model.Availability, error) {
	args := m.Called(ctx, slotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Availability), args.Error(1)
}

func (m *MockDoctorRepository) ListAvailability(ctx context.Context, doctorID uuid.UUID, startTime time.Time, endTime time.Time, isBooked *bool) ([]*model.Availability, error) {
	args := m.Called(ctx, doctorID, startTime, endTime, isBooked)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Availability), args.Error(1)
}

func (m *MockDoctorRepository) CheckOverlappingSlot(ctx context.Context, doctorID uuid.UUID, startTime time.Time, endTime time.Time) (bool, error) {
	args := m.Called(ctx, doctorID, startTime, endTime)
	return args.Bool(0), args.Error(1)
}

type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) Log(ctx context.Context, entry shared.AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func TestDoctorService_GetProfileByUserID(t *testing.T) {
	mockRepo := new(MockDoctorRepository)
	svc := NewDoctorService(mockRepo, nil, nil)

	userID := uuid.New()
	dID := uuid.New()
	license := "123.456"
	specID := uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")
	phone := "+6281122334455"

	doctor := &model.Doctor{
		ID:                   dID,
		UserID:               userID,
		Email:                "amir@doctor.com",
		FullName:             "Dr. Amir",
		PhoneNumber:          &phone,
		SpecialtyID:          &specID,
		LicenseNumber:        &license,
		IsCredentialVerified: true,
		ConsultationFee:      150000,
	}

	t.Run("Success doctor retrieval", func(t *testing.T) {
		mockRepo.On("GetByUserID", mock.Anything, userID).Return(doctor, nil).Once()

		resp, err := svc.GetProfileByUserID(context.Background(), userID)
		require.NoError(t, err)
		assert.Equal(t, dID.String(), resp.ID)
		assert.Equal(t, specID.String(), *resp.SpecialtyID)
		assert.True(t, resp.IsCredentialVerified)
		mockRepo.AssertExpectations(t)
	})
}

func TestDoctorService_UpdateProfile(t *testing.T) {
	mockRepo := new(MockDoctorRepository)
	svc := NewDoctorService(mockRepo, nil, nil)

	userID := uuid.New()
	dID := uuid.New()
	phoneBefore := "+628111111111"
	doctorBefore := &model.Doctor{
		ID:          dID,
		UserID:      userID,
		Email:       "amir@doctor.com",
		FullName:    "Dr. Amir",
		PhoneNumber: &phoneBefore,
	}

	t.Run("Success Update", func(t *testing.T) {
		req := dto.UpdateDoctorRequest{
			SpecialtyID:     "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			LicenseNumber:   "987.654",
			ConsultationFee: 200000,
			PhoneNumber:     "+6281122334455",
		}

		mockRepo.On("GetByUserID", mock.Anything, userID).Return(doctorBefore, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Once()

		resp, err := svc.UpdateProfile(context.Background(), userID, req)
		require.NoError(t, err)
		assert.Equal(t, "f47ac10b-58cc-4372-a567-0e02b2c3d479", *resp.SpecialtyID)
		assert.Equal(t, "987.654", *resp.LicenseNumber)
		assert.Equal(t, int64(200000), resp.ConsultationFee)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Validation error", func(t *testing.T) {
		req := dto.UpdateDoctorRequest{
			SpecialtyID:     "",
			LicenseNumber:   "",
			ConsultationFee: -100,
			PhoneNumber:     "invalid-phone",
		}

		resp, err := svc.UpdateProfile(context.Background(), userID, req)
		assert.Nil(t, resp)
		var valErrs validator.ValidationErrors
		assert.True(t, errors.As(err, &valErrs))
		assert.Len(t, valErrs, 4)
	})
}

func TestDoctorService_VerifyDoctor(t *testing.T) {
	mockRepo := new(MockDoctorRepository)
	mockAudit := new(MockAuditService)
	svc := NewDoctorService(mockRepo, nil, mockAudit)

	adminID := uuid.New()
	doctorID := uuid.New()

	t.Run("Success verify with audit log", func(t *testing.T) {
		mockRepo.On("Verify", mock.Anything, doctorID).Return(nil).Once()
		mockAudit.On("Log", mock.Anything, mock.MatchedBy(func(entry shared.AuditEntry) bool {
			return entry.ActorID == adminID && entry.Action == "doctor.verified" && entry.TargetID == doctorID
		})).Return(nil).Once()

		err := svc.VerifyDoctor(context.Background(), adminID, doctorID, "127.0.0.1", "Mozilla")
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockAudit.AssertExpectations(t)
	})
}
