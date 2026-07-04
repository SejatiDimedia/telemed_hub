package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	appointmentDto "github.com/timurdianradhasejati/telemed_hub/internal/appointment/dto"
	appointmentService "github.com/timurdianradhasejati/telemed_hub/internal/appointment/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/model"
)

type MockConsultationRepository struct {
	mock.Mock
}

func (m *MockConsultationRepository) Create(ctx context.Context, cons *model.Consultation) error {
	args := m.Called(ctx, cons)
	return args.Error(0)
}

func (m *MockConsultationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Consultation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Consultation), args.Error(1)
}

func (m *MockConsultationRepository) GetByAppointmentID(ctx context.Context, aptID uuid.UUID) (*model.Consultation, error) {
	args := m.Called(ctx, aptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Consultation), args.Error(1)
}

func (m *MockConsultationRepository) Update(ctx context.Context, cons *model.Consultation) error {
	args := m.Called(ctx, cons)
	return args.Error(0)
}

type MockAppointmentService struct {
	mock.Mock
}

func (m *MockAppointmentService) Book(ctx context.Context, patientUserID uuid.UUID, req appointmentDto.CreateAppointmentRequest) (*appointmentDto.AppointmentResponse, error) {
	args := m.Called(ctx, patientUserID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appointmentDto.AppointmentResponse), args.Error(1)
}

func (m *MockAppointmentService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*appointmentDto.AppointmentResponse, error) {
	args := m.Called(ctx, id, userID, roles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appointmentDto.AppointmentResponse), args.Error(1)
}

func (m *MockAppointmentService) List(ctx context.Context, userID uuid.UUID, roles []string, statusFilter string) ([]*appointmentDto.AppointmentResponse, error) {
	args := m.Called(ctx, userID, roles, statusFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*appointmentDto.AppointmentResponse), args.Error(1)
}

func (m *MockAppointmentService) Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID, req appointmentDto.CancelAppointmentRequest) error {
	args := m.Called(ctx, id, userID, req)
	return args.Error(0)
}

func (m *MockAppointmentService) Reschedule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req appointmentDto.RescheduleAppointmentRequest) (*appointmentDto.AppointmentResponse, error) {
	args := m.Called(ctx, id, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appointmentDto.AppointmentResponse), args.Error(1)
}

func (m *MockAppointmentService) SetConsultationService(consSvc appointmentService.ConsultationServiceClient) {
	m.Called(consSvc)
}

func TestConsultationService_Start(t *testing.T) {
	mockRepo := new(MockConsultationRepository)
	mockApt := new(MockAppointmentService)
	svc := NewConsultationService(mockRepo, mockApt)

	ctx := context.Background()
	consID := uuid.New()
	aptID := uuid.New()
	doctorUserID := uuid.New()

	t.Run("Success starting consultation", func(t *testing.T) {
		cons := &model.Consultation{
			ID:            consID,
			AppointmentID: aptID,
			Status:        "scheduled",
		}
		mockRepo.On("GetByID", ctx, consID).Return(cons, nil).Once()

		aptResp := &appointmentDto.AppointmentResponse{
			ID:       aptID.String(),
			DoctorID: uuid.New().String(),
		}
		mockApt.On("GetByID", ctx, aptID, doctorUserID, []string{"doctor"}).Return(aptResp, nil).Once()

		mockRepo.On("Update", ctx, mock.MatchedBy(func(c *model.Consultation) bool {
			return c.Status == "in_progress" && c.StartedAt != nil
		})).Return(nil).Once()

		resp, err := svc.Start(ctx, consID, doctorUserID)
		assert.NoError(t, err)
		assert.Equal(t, "in_progress", resp.Status)
		assert.NotNil(t, resp.StartedAt)

		mockRepo.AssertExpectations(t)
		mockApt.AssertExpectations(t)
	})

	t.Run("Invalid state transition error", func(t *testing.T) {
		cons := &model.Consultation{
			ID:            consID,
			AppointmentID: aptID,
			Status:        "completed",
		}
		mockRepo.On("GetByID", ctx, consID).Return(cons, nil).Once()

		resp, err := svc.Start(ctx, consID, doctorUserID)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidTransition)

		mockRepo.AssertExpectations(t)
	})
}

func TestConsultationService_Complete(t *testing.T) {
	mockRepo := new(MockConsultationRepository)
	mockApt := new(MockAppointmentService)
	svc := NewConsultationService(mockRepo, mockApt)

	ctx := context.Background()
	consID := uuid.New()
	aptID := uuid.New()
	doctorUserID := uuid.New()

	t.Run("Success completing consultation", func(t *testing.T) {
		cons := &model.Consultation{
			ID:            consID,
			AppointmentID: aptID,
			Status:        "in_progress",
		}
		mockRepo.On("GetByID", ctx, consID).Return(cons, nil).Once()

		aptResp := &appointmentDto.AppointmentResponse{
			ID: aptID.String(),
		}
		mockApt.On("GetByID", ctx, aptID, doctorUserID, []string{"doctor"}).Return(aptResp, nil).Once()

		mockRepo.On("Update", ctx, mock.MatchedBy(func(c *model.Consultation) bool {
			return c.Status == "completed" && c.EndedAt != nil
		})).Return(nil).Once()

		resp, err := svc.Complete(ctx, consID, doctorUserID)
		assert.NoError(t, err)
		assert.Equal(t, "completed", resp.Status)

		mockRepo.AssertExpectations(t)
		mockApt.AssertExpectations(t)
	})
}
