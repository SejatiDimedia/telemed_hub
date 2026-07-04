package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	consultationDTO "github.com/timurdianradhasejati/telemed_hub/internal/consultation/dto"
	doctorDTO "github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	patientDTO "github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/repository"
)

// --- Mocks ---

type MockPrescriptionRepository struct{ mock.Mock }

func (m *MockPrescriptionRepository) Create(ctx context.Context, pres *model.Prescription) error {
	args := m.Called(ctx, pres)
	return args.Error(0)
}

func (m *MockPrescriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Prescription, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Prescription), args.Error(1)
}

func (m *MockPrescriptionRepository) ListByPatientID(ctx context.Context, patientID uuid.UUID) ([]*model.Prescription, error) {
	args := m.Called(ctx, patientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Prescription), args.Error(1)
}

func (m *MockPrescriptionRepository) ListByDoctorID(ctx context.Context, doctorID uuid.UUID) ([]*model.Prescription, error) {
	args := m.Called(ctx, doctorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Prescription), args.Error(1)
}

// Static compile-time check
var _ repository.PrescriptionRepository = (*MockPrescriptionRepository)(nil)

type MockConsultationService struct{ mock.Mock }

func (m *MockConsultationService) CreateConsultation(ctx context.Context, appointmentID uuid.UUID) error {
	return m.Called(ctx, appointmentID).Error(0)
}
func (m *MockConsultationService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*consultationDTO.ConsultationResponse, error) {
	args := m.Called(ctx, id, userID, roles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*consultationDTO.ConsultationResponse), args.Error(1)
}
func (m *MockConsultationService) Start(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID) (*consultationDTO.ConsultationResponse, error) {
	args := m.Called(ctx, id, doctorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*consultationDTO.ConsultationResponse), args.Error(1)
}
func (m *MockConsultationService) Complete(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID) (*consultationDTO.ConsultationResponse, error) {
	args := m.Called(ctx, id, doctorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*consultationDTO.ConsultationResponse), args.Error(1)
}
func (m *MockConsultationService) UpdateNotes(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID, notes string) (*consultationDTO.ConsultationResponse, error) {
	args := m.Called(ctx, id, doctorUserID, notes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*consultationDTO.ConsultationResponse), args.Error(1)
}
func (m *MockConsultationService) CancelConsultation(ctx context.Context, appointmentID uuid.UUID) error {
	return m.Called(ctx, appointmentID).Error(0)
}

type MockDoctorService struct{ mock.Mock }

func (m *MockDoctorService) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*doctorDTO.DoctorResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*doctorDTO.DoctorResponse), args.Error(1)
}
func (m *MockDoctorService) GetProfileByID(ctx context.Context, id uuid.UUID) (*doctorDTO.DoctorResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*doctorDTO.DoctorResponse), args.Error(1)
}
func (m *MockDoctorService) UpdateProfile(ctx context.Context, userID uuid.UUID, req doctorDTO.UpdateDoctorRequest) (*doctorDTO.DoctorResponse, error) {
	return nil, nil
}
func (m *MockDoctorService) VerifyDoctor(ctx context.Context, adminUserID uuid.UUID, doctorID uuid.UUID, ip string, ua string) error {
	return nil
}
func (m *MockDoctorService) ListDoctors(ctx context.Context, specialty *string, onlyVerified bool, sortBy string, order string, page int, limit int) ([]*doctorDTO.DoctorResponse, int, error) {
	return nil, 0, nil
}
func (m *MockDoctorService) AddAvailability(ctx context.Context, doctorUserID uuid.UUID, req doctorDTO.CreateAvailabilityRequest) (*doctorDTO.AvailabilityResponse, error) {
	return nil, nil
}
func (m *MockDoctorService) RemoveAvailability(ctx context.Context, doctorUserID uuid.UUID, slotID uuid.UUID) error {
	return nil
}
func (m *MockDoctorService) GetAvailability(ctx context.Context, doctorID uuid.UUID, startTimeStr, endTimeStr string, isBooked *bool) ([]*doctorDTO.AvailabilityResponse, error) {
	return nil, nil
}

type MockPatientService struct{ mock.Mock }

func (m *MockPatientService) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*patientDTO.PatientResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*patientDTO.PatientResponse), args.Error(1)
}
func (m *MockPatientService) GetProfileByID(ctx context.Context, id uuid.UUID) (*patientDTO.PatientResponse, error) {
	return nil, nil
}
func (m *MockPatientService) UpdateProfile(ctx context.Context, userID uuid.UUID, req patientDTO.UpdatePatientRequest) (*patientDTO.PatientResponse, error) {
	return nil, nil
}

// --- Helpers ---

func newTestService(
	repo *MockPrescriptionRepository,
	consSvc *MockConsultationService,
	docSvc *MockDoctorService,
	patSvc *MockPatientService,
) *PrescriptionServiceImpl {
	return NewPrescriptionService(repo, consSvc, docSvc, patSvc)
}

// --- Tests ---

func Test_IssuePrescription_Success(t *testing.T) {
	ctx := context.Background()
	doctorUserID := uuid.New()
	doctorID := uuid.New()
	patientID := uuid.New()
	consultationID := uuid.New()
	medicineID := uuid.New()
	prescriptionID := uuid.New()

	repo := new(MockPrescriptionRepository)
	consSvc := new(MockConsultationService)
	docSvc := new(MockDoctorService)
	patSvc := new(MockPatientService)

	docSvc.On("GetProfileByUserID", ctx, doctorUserID).Return(&doctorDTO.DoctorResponse{
		ID:     doctorID.String(),
		UserID: doctorUserID.String(),
	}, nil)

	consSvc.On("GetByID", ctx, consultationID, doctorUserID, []string{"doctor"}).Return(&consultationDTO.ConsultationResponse{
		ID:            consultationID.String(),
		AppointmentID: uuid.New().String(),
		DoctorID:      doctorID.String(), // same as the calling doctor
		PatientID:     patientID.String(),
		Status:        "in_progress",
	}, nil)

	repo.On("Create", ctx, mock.AnythingOfType("*model.Prescription")).Return(nil)

	repo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(&model.Prescription{
		ID:             prescriptionID,
		ConsultationID: consultationID,
		PatientID:      patientID,
		DoctorID:       doctorID,
		Status:         "active",
		Items: []model.PrescriptionItem{
			{
				ID:           uuid.New(),
				MedicineID:   medicineID,
				MedicineName: "Amoxicillin 500mg",
				Dosage:       "500mg twice daily",
				Quantity:     10,
			},
		},
	}, nil)

	svc := newTestService(repo, consSvc, docSvc, patSvc)
	req := dto.CreatePrescriptionRequest{
		ConsultationID: consultationID.String(),
		Items: []dto.PrescriptionItemRequest{
			{
				MedicineID: medicineID.String(),
				Dosage:     "500mg twice daily",
				Quantity:   10,
			},
		},
	}

	resp, err := svc.Issue(ctx, doctorUserID, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "active", resp.Status)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Amoxicillin 500mg", resp.Items[0].MedicineName)

	repo.AssertExpectations(t)
	consSvc.AssertExpectations(t)
	docSvc.AssertExpectations(t)
}

func Test_IssuePrescription_FailsWhenConsultationScheduled(t *testing.T) {
	ctx := context.Background()
	doctorUserID := uuid.New()
	doctorID := uuid.New()
	consultationID := uuid.New()

	repo := new(MockPrescriptionRepository)
	consSvc := new(MockConsultationService)
	docSvc := new(MockDoctorService)
	patSvc := new(MockPatientService)

	docSvc.On("GetProfileByUserID", ctx, doctorUserID).Return(&doctorDTO.DoctorResponse{
		ID:     doctorID.String(),
		UserID: doctorUserID.String(),
	}, nil)

	consSvc.On("GetByID", ctx, consultationID, doctorUserID, []string{"doctor"}).Return(&consultationDTO.ConsultationResponse{
		ID:       consultationID.String(),
		DoctorID: doctorID.String(),
		Status:   "scheduled", // must be in_progress or completed
	}, nil)

	svc := newTestService(repo, consSvc, docSvc, patSvc)
	req := dto.CreatePrescriptionRequest{
		ConsultationID: consultationID.String(),
		Items: []dto.PrescriptionItemRequest{
			{MedicineID: uuid.New().String(), Dosage: "10mg", Quantity: 5},
		},
	}

	_, err := svc.Issue(ctx, doctorUserID, req)
	assert.ErrorIs(t, err, ErrInvalidConsultationStatus)
	repo.AssertNotCalled(t, "Create")
}

func Test_IssuePrescription_FailsWhenNotAssignedDoctor(t *testing.T) {
	ctx := context.Background()
	callerUserID := uuid.New()
	callerDoctorID := uuid.New()
	assignedDoctorID := uuid.New() // a different doctor's profile ID
	consultationID := uuid.New()

	repo := new(MockPrescriptionRepository)
	consSvc := new(MockConsultationService)
	docSvc := new(MockDoctorService)
	patSvc := new(MockPatientService)

	docSvc.On("GetProfileByUserID", ctx, callerUserID).Return(&doctorDTO.DoctorResponse{
		ID:     callerDoctorID.String(), // caller's doctor ID
		UserID: callerUserID.String(),
	}, nil)

	consSvc.On("GetByID", ctx, consultationID, callerUserID, []string{"doctor"}).Return(&consultationDTO.ConsultationResponse{
		ID:       consultationID.String(),
		DoctorID: assignedDoctorID.String(), // a different doctor owns it
		Status:   "in_progress",
	}, nil)

	svc := newTestService(repo, consSvc, docSvc, patSvc)
	req := dto.CreatePrescriptionRequest{
		ConsultationID: consultationID.String(),
		Items: []dto.PrescriptionItemRequest{
			{MedicineID: uuid.New().String(), Dosage: "10mg", Quantity: 5},
		},
	}

	_, err := svc.Issue(ctx, callerUserID, req)
	assert.ErrorIs(t, err, ErrUnauthorized)
	repo.AssertNotCalled(t, "Create")
}

func Test_GetByID_PatientCanSeeOwnPrescription(t *testing.T) {
	ctx := context.Background()
	patientUserID := uuid.New()
	patientID := uuid.New()
	prescriptionID := uuid.New()

	repo := new(MockPrescriptionRepository)
	consSvc := new(MockConsultationService)
	docSvc := new(MockDoctorService)
	patSvc := new(MockPatientService)

	repo.On("GetByID", ctx, prescriptionID).Return(&model.Prescription{
		ID:        prescriptionID,
		PatientID: patientID,
		DoctorID:  uuid.New(),
		Status:    "active",
		Items:     []model.PrescriptionItem{},
	}, nil)

	patSvc.On("GetProfileByUserID", ctx, patientUserID).Return(&patientDTO.PatientResponse{
		ID:     patientID.String(),
		UserID: patientUserID.String(),
	}, nil)

	svc := newTestService(repo, consSvc, docSvc, patSvc)
	resp, err := svc.GetByID(ctx, prescriptionID, patientUserID, []string{"patient"})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, prescriptionID.String(), resp.ID)
}

func Test_GetByID_PatientCannotSeeOtherPatientPrescription(t *testing.T) {
	ctx := context.Background()
	patientUserID := uuid.New()
	myPatientID := uuid.New()
	otherPatientID := uuid.New() // prescription belongs to someone else
	prescriptionID := uuid.New()

	repo := new(MockPrescriptionRepository)
	consSvc := new(MockConsultationService)
	docSvc := new(MockDoctorService)
	patSvc := new(MockPatientService)

	repo.On("GetByID", ctx, prescriptionID).Return(&model.Prescription{
		ID:        prescriptionID,
		PatientID: otherPatientID, // different patient
		DoctorID:  uuid.New(),
		Status:    "active",
	}, nil)

	patSvc.On("GetProfileByUserID", ctx, patientUserID).Return(&patientDTO.PatientResponse{
		ID:     myPatientID.String(),
		UserID: patientUserID.String(),
	}, nil)

	svc := newTestService(repo, consSvc, docSvc, patSvc)
	_, err := svc.GetByID(ctx, prescriptionID, patientUserID, []string{"patient"})
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func Test_ListPrescriptions_DoctorSeesOwnIssued(t *testing.T) {
	ctx := context.Background()
	doctorUserID := uuid.New()
	doctorID := uuid.New()

	repo := new(MockPrescriptionRepository)
	consSvc := new(MockConsultationService)
	docSvc := new(MockDoctorService)
	patSvc := new(MockPatientService)

	docSvc.On("GetProfileByUserID", ctx, doctorUserID).Return(&doctorDTO.DoctorResponse{
		ID:     doctorID.String(),
		UserID: doctorUserID.String(),
	}, nil)

	repo.On("ListByDoctorID", ctx, doctorID).Return([]*model.Prescription{
		{ID: uuid.New(), DoctorID: doctorID, Status: "active", Items: []model.PrescriptionItem{}},
	}, nil)

	svc := newTestService(repo, consSvc, docSvc, patSvc)
	records, err := svc.List(ctx, doctorUserID, []string{"doctor"})
	assert.NoError(t, err)
	assert.Len(t, records, 1)
}
