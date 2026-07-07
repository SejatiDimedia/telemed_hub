package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	patientDto "github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
	patientSvc "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/shared"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/repository"
)

type MockMedicalRecordRepository struct {
	mock.Mock
}

func (m *MockMedicalRecordRepository) Create(ctx context.Context, rec *model.MedicalRecord) error {
	args := m.Called(ctx, rec)
	return args.Error(0)
}

func (m *MockMedicalRecordRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MedicalRecord, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MedicalRecord), args.Error(1)
}

func (m *MockMedicalRecordRepository) List(ctx context.Context, filter dto.ListMedicalRecordsFilter) ([]*model.MedicalRecord, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MedicalRecord), args.Error(1)
}

func (m *MockMedicalRecordRepository) Update(ctx context.Context, rec *model.MedicalRecord) error {
	args := m.Called(ctx, rec)
	return args.Error(0)
}

func (m *MockMedicalRecordRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	args := m.Called(ctx, id, deletedBy)
	return args.Error(0)
}

func (m *MockMedicalRecordRepository) HasTreatmentRelationship(ctx context.Context, doctorUserID, patientID uuid.UUID) (bool, error) {
	args := m.Called(ctx, doctorUserID, patientID)
	return args.Bool(0), args.Error(1)
}

var _ repository.MedicalRecordRepository = (*MockMedicalRecordRepository)(nil)

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

var _ patientSvc.PatientService = (*MockPatientService)(nil)


type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) Log(ctx context.Context, entry shared.AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

var _ shared.AuditService = (*MockAuditService)(nil)

func TestMedicalRecordService_Create(t *testing.T) {
	mockRepo := new(MockMedicalRecordRepository)
	mockPatientSvc := new(MockPatientService)
	mockAuditSvc := new(MockAuditService)

	svc := NewMedicalRecordService(mockRepo, mockPatientSvc, mockAuditSvc)

	doctorUserID := uuid.New()
	patientID := uuid.New()
	req := dto.CreateMedicalRecordRequest{
		PatientID:  patientID.String(),
		RecordType: "diagnosis",
		Content:    "Flu",
	}

	t.Run("Create success as doctor with treatment relationship", func(t *testing.T) {
		mockRepo.On("HasTreatmentRelationship", mock.Anything, doctorUserID, patientID).Return(true, nil).Once()
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
		mockAuditSvc.On("Log", mock.Anything, mock.MatchedBy(func(e shared.AuditEntry) bool {
			return e.Action == "medical_record.created" && e.ActorID == doctorUserID
		})).Return(nil).Once()

		resp, err := svc.Create(context.Background(), doctorUserID, []string{"doctor"}, "127.0.0.1", "curl", req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockRepo.AssertExpectations(t)
		mockAuditSvc.AssertExpectations(t)
	})

	t.Run("Create fails if doctor has no treatment relationship", func(t *testing.T) {
		mockRepo.On("HasTreatmentRelationship", mock.Anything, doctorUserID, patientID).Return(false, nil).Once()

		_, err := svc.Create(context.Background(), doctorUserID, []string{"doctor"}, "127.0.0.1", "curl", req)
		assert.ErrorIs(t, err, ErrTreatmentRelationshipRequired)
		mockRepo.AssertExpectations(t)
	})
}

func TestMedicalRecordService_GetByID(t *testing.T) {
	mockRepo := new(MockMedicalRecordRepository)
	mockPatientSvc := new(MockPatientService)
	mockAuditSvc := new(MockAuditService)

	svc := NewMedicalRecordService(mockRepo, mockPatientSvc, mockAuditSvc)

	doctorUserID := uuid.New()
	patientID := uuid.New()
	recordID := uuid.New()

	rec := &model.MedicalRecord{
		ID:         recordID,
		PatientID:  patientID,
		RecordType: "allergy",
		Content:    "Dust",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	t.Run("GetByID success for doctor writes view audit log", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, recordID).Return(rec, nil).Once()
		mockRepo.On("HasTreatmentRelationship", mock.Anything, doctorUserID, patientID).Return(true, nil).Once()
		mockAuditSvc.On("Log", mock.Anything, mock.MatchedBy(func(e shared.AuditEntry) bool {
			return e.Action == "medical_record.viewed" && e.ActorID == doctorUserID && e.TargetID == recordID
		})).Return(nil).Once()

		resp, err := svc.GetByID(context.Background(), doctorUserID, []string{"doctor"}, "127.0.0.1", "curl", recordID)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockRepo.AssertExpectations(t)
		mockAuditSvc.AssertExpectations(t)
	})

	t.Run("GetByID for patient owner does not write audit log", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, recordID).Return(rec, nil).Once()
		patientUserID := uuid.New()
		mockPatientSvc.On("GetProfileByUserID", mock.Anything, patientUserID).Return(&patientDto.PatientResponse{
			ID: patientID.String(),
		}, nil).Once()

		// Notice that mockAuditSvc.Log is NOT mocked or expected to be called!
		resp, err := svc.GetByID(context.Background(), patientUserID, []string{"patient"}, "127.0.0.1", "curl", recordID)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockRepo.AssertExpectations(t)
		mockPatientSvc.AssertExpectations(t)
		mockAuditSvc.AssertExpectations(t) // Asserts Log was not called
	})
}
