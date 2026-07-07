package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/model"
	doctorDto "github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	patientDto "github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet"
	walletDto "github.com/timurdianradhasejati/telemed_hub/internal/wallet/dto"
	"github.com/jackc/pgx/v5"
)

type MockAppointmentRepository struct {
	mock.Mock
}

func (m *MockAppointmentRepository) CreateWithLock(ctx context.Context, apt *model.Appointment) error {
	args := m.Called(ctx, apt)
	return args.Error(0)
}

func (m *MockAppointmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Appointment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Appointment), args.Error(1)
}

func (m *MockAppointmentRepository) List(ctx context.Context, filter map[string]any) ([]*model.Appointment, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Appointment), args.Error(1)
}

func (m *MockAppointmentRepository) UpdateStatusWithLock(ctx context.Context, id uuid.UUID, status string, cancelReason *string) error {
	args := m.Called(ctx, id, status, cancelReason)
	return args.Error(0)
}

func (m *MockAppointmentRepository) RescheduleWithLock(ctx context.Context, id uuid.UUID, newAvailabilityID uuid.UUID, scheduledAt time.Time) error {
	args := m.Called(ctx, id, newAvailabilityID, scheduledAt)
	return args.Error(0)
}

func (m *MockAppointmentRepository) GetAvailabilityByID(ctx context.Context, id uuid.UUID) (bool, uuid.UUID, time.Time, time.Time, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Get(1).(uuid.UUID), args.Get(2).(time.Time), args.Get(3).(time.Time), args.Error(4)
}

// Stubs for doctor/patient/wallet
type MockPatientService struct {
	mock.Mock
}

func (m *MockPatientService) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*patientDto.PatientResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*patientDto.PatientResponse), args.Error(1)
}

func (m *MockPatientService) GetProfileByID(ctx context.Context, id uuid.UUID) (*patientDto.PatientResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*patientDto.PatientResponse), args.Error(1)
}

func (m *MockPatientService) UpdateProfile(ctx context.Context, userID uuid.UUID, req patientDto.UpdatePatientRequest) (*patientDto.PatientResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*patientDto.PatientResponse), args.Error(1)
}

type MockDoctorService struct {
	mock.Mock
}

func (m *MockDoctorService) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*doctorDto.DoctorResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*doctorDto.DoctorResponse), args.Error(1)
}

func (m *MockDoctorService) GetProfileByID(ctx context.Context, id uuid.UUID) (*doctorDto.DoctorResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*doctorDto.DoctorResponse), args.Error(1)
}

func (m *MockDoctorService) UpdateProfile(ctx context.Context, userID uuid.UUID, req doctorDto.UpdateDoctorRequest) (*doctorDto.DoctorResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*doctorDto.DoctorResponse), args.Error(1)
}

func (m *MockDoctorService) VerifyDoctor(ctx context.Context, adminUserID uuid.UUID, doctorID uuid.UUID, ipAddress string, userAgent string) error {
	args := m.Called(ctx, adminUserID, doctorID, ipAddress, userAgent)
	return args.Error(0)
}

func (m *MockDoctorService) ListDoctors(ctx context.Context, specialty *string, onlyVerified bool, sortBy string, order string, page int, limit int) ([]*doctorDto.DoctorResponse, int, error) {
	args := m.Called(ctx, specialty, onlyVerified, sortBy, order, page, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*doctorDto.DoctorResponse), args.Int(1), args.Error(2)
}

func (m *MockDoctorService) AddAvailability(ctx context.Context, doctorUserID uuid.UUID, req doctorDto.CreateAvailabilityRequest) (*doctorDto.AvailabilityResponse, error) {
	args := m.Called(ctx, doctorUserID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*doctorDto.AvailabilityResponse), args.Error(1)
}

func (m *MockDoctorService) RemoveAvailability(ctx context.Context, doctorUserID uuid.UUID, slotID uuid.UUID) error {
	args := m.Called(ctx, doctorUserID, slotID)
	return args.Error(0)
}

func (m *MockDoctorService) GetAvailability(ctx context.Context, doctorID uuid.UUID, startTimeStr, endTimeStr string, isBooked *bool) ([]*doctorDto.AvailabilityResponse, error) {
	args := m.Called(ctx, doctorID, startTimeStr, endTimeStr, isBooked)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*doctorDto.AvailabilityResponse), args.Error(1)
}

type MockWalletService struct {
	mock.Mock
}

func (m *MockWalletService) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWalletService) Deduct(ctx context.Context, userID uuid.UUID, amount int64, description string) error {
	args := m.Called(ctx, userID, amount, description)
	return args.Error(0)
}

func (m *MockWalletService) Refund(ctx context.Context, userID uuid.UUID, amount int64, description string) error {
	args := m.Called(ctx, userID, amount, description)
	return args.Error(0)
}

func (m *MockWalletService) GetBalanceDetails(ctx context.Context, userID uuid.UUID) (*walletDto.WalletResponse, error) {
	return nil, nil
}

func (m *MockWalletService) TopUp(ctx context.Context, userID uuid.UUID, amount float64, idempotencyKey *string) (*walletDto.TransactionResponse, error) {
	return nil, nil
}

func (m *MockWalletService) ListTransactions(ctx context.Context, userID uuid.UUID, typeFilter *string, page, limit int) ([]*walletDto.TransactionResponse, int, error) {
	return nil, 0, nil
}

func (m *MockWalletService) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*walletDto.TransactionResponse, error) {
	return nil, nil
}

func (m *MockWalletService) DeductTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, amount int64, description string, idempotencyKey *string) error {
	args := m.Called(ctx, tx, userID, amount, description, idempotencyKey)
	return args.Error(0)
}

func (m *MockWalletService) RefundTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, amount int64, description string) error {
	args := m.Called(ctx, tx, userID, amount, description)
	return args.Error(0)
}

var _ wallet.WalletService = (*MockWalletService)(nil)

func TestAppointmentService_Book(t *testing.T) {
	mockRepo := new(MockAppointmentRepository)
	mockPatient := new(MockPatientService)
	mockDoctor := new(MockDoctorService)
	mockWallet := new(MockWalletService)
	svc := NewAppointmentService(mockRepo, mockPatient, mockDoctor, mockWallet, 60)

	ctx := context.Background()
	patientUserID := uuid.New()
	patientID := uuid.New()
	doctorID := uuid.New()
	availID := uuid.New()

	t.Run("Incomplete patient profile check", func(t *testing.T) {
		dob := ""
		gender := ""
		patResp := &patientDto.PatientResponse{
			ID:          patientID.String(),
			UserID:      patientUserID.String(),
			DateOfBirth: &dob,
			Gender:      &gender,
		}
		mockPatient.On("GetProfileByUserID", ctx, patientUserID).Return(patResp, nil).Once()

		req := dto.CreateAppointmentRequest{
			DoctorID:       doctorID.String(),
			AvailabilityID: availID.String(),
		}

		resp, err := svc.Book(ctx, patientUserID, req)
		assert.Nil(t, resp)
		assert.Equal(t, ErrProfileIncomplete, err)
		mockPatient.AssertExpectations(t)
	})

	t.Run("Doctor not verified check", func(t *testing.T) {
		dob := "1995-05-15"
		gender := "male"
		patResp := &patientDto.PatientResponse{
			ID:          patientID.String(),
			UserID:      patientUserID.String(),
			DateOfBirth: &dob,
			Gender:      &gender,
		}
		mockPatient.On("GetProfileByUserID", ctx, patientUserID).Return(patResp, nil).Once()

		docResp := &doctorDto.DoctorResponse{
			ID:                   doctorID.String(),
			IsCredentialVerified: false,
		}
		mockDoctor.On("GetProfileByID", ctx, doctorID).Return(docResp, nil).Once()

		req := dto.CreateAppointmentRequest{
			DoctorID:       doctorID.String(),
			AvailabilityID: availID.String(),
		}

		resp, err := svc.Book(ctx, patientUserID, req)
		assert.Nil(t, resp)
		assert.Equal(t, ErrDoctorNotVerified, err)
		mockPatient.AssertExpectations(t)
		mockDoctor.AssertExpectations(t)
	})

	t.Run("Insufficient wallet balance check", func(t *testing.T) {
		dob := "1995-05-15"
		gender := "male"
		patResp := &patientDto.PatientResponse{
			ID:          patientID.String(),
			UserID:      patientUserID.String(),
			DateOfBirth: &dob,
			Gender:      &gender,
		}
		mockPatient.On("GetProfileByUserID", ctx, patientUserID).Return(patResp, nil).Once()

		docResp := &doctorDto.DoctorResponse{
			ID:                   doctorID.String(),
			IsCredentialVerified: true,
			ConsultationFee:      150000,
		}
		mockDoctor.On("GetProfileByID", ctx, doctorID).Return(docResp, nil).Once()

		startTime := time.Now().Add(24 * time.Hour).UTC()
		endTime := startTime.Add(30 * time.Minute)
		mockRepo.On("GetAvailabilityByID", ctx, availID).Return(false, doctorID, startTime, endTime, nil).Once()

		mockWallet.On("Deduct", ctx, patientUserID, int64(150000), mock.Anything).Return(wallet.ErrInsufficientBalance).Once()

		req := dto.CreateAppointmentRequest{
			DoctorID:       doctorID.String(),
			AvailabilityID: availID.String(),
		}

		resp, err := svc.Book(ctx, patientUserID, req)
		assert.Nil(t, resp)
		assert.Equal(t, ErrInsufficientBalance, err)

		mockPatient.AssertExpectations(t)
		mockDoctor.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
		mockWallet.AssertExpectations(t)
	})
}
