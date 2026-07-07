package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/validator"
	doctorSvc "github.com/timurdianradhasejati/telemed_hub/internal/doctor/service"
	patientSvc "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification"
)

type AppointmentServiceImpl struct {
	repo            repository.AppointmentRepository
	patientSvc      patientSvc.PatientService
	doctorSvc       doctorSvc.DoctorService
	walletSvc       wallet.WalletService
	notificationSvc notification.NotificationService
	consultationSvc ConsultationServiceClient
	cutoffMinutes   int
}

func NewAppointmentService(
	repo repository.AppointmentRepository,
	patientSvc patientSvc.PatientService,
	doctorSvc doctorSvc.DoctorService,
	walletSvc wallet.WalletService,
	notificationSvc notification.NotificationService,
	cutoffMinutes int,
) *AppointmentServiceImpl {
	if cutoffMinutes <= 0 {
		cutoffMinutes = 60 // default 60 minutes
	}
	return &AppointmentServiceImpl{
		repo:            repo,
		patientSvc:      patientSvc,
		doctorSvc:       doctorSvc,
		walletSvc:       walletSvc,
		notificationSvc: notificationSvc,
		cutoffMinutes:   cutoffMinutes,
	}
}

func (s *AppointmentServiceImpl) SetConsultationService(consSvc ConsultationServiceClient) {
	s.consultationSvc = consSvc
}

func (s *AppointmentServiceImpl) Book(ctx context.Context, patientUserID uuid.UUID, req dto.CreateAppointmentRequest) (*dto.AppointmentResponse, error) {
	// 1. Validate request IDs
	docUUID, availUUID, err := validator.ValidateCreateAppointment(req)
	if err != nil {
		return nil, err
	}

	// 2. Fetch patient and enforce complete profile gate
	patProfile, err := s.patientSvc.GetProfileByUserID(ctx, patientUserID)
	if err != nil {
		return nil, err
	}
	if patProfile.DateOfBirth == nil || *patProfile.DateOfBirth == "" || patProfile.Gender == nil || *patProfile.Gender == "" {
		return nil, ErrProfileIncomplete
	}

	patUUID, err := uuid.Parse(patProfile.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid patient profile uuid: %w", err)
	}

	// 3. Fetch doctor and enforce credential verification gate
	docProfile, err := s.doctorSvc.GetProfileByID(ctx, docUUID)
	if err != nil {
		return nil, err
	}
	if !docProfile.IsCredentialVerified {
		return nil, ErrDoctorNotVerified
	}

	// 4. Fetch availability slot details
	isBooked, slotDocID, startTime, _, err := s.repo.GetAvailabilityByID(ctx, availUUID)
	if err != nil {
		if errors.Is(err, repository.ErrAvailabilityNotFound) {
			return nil, repository.ErrAvailabilityNotFound
		}
		return nil, err
	}

	if slotDocID != docUUID {
		return nil, errors.New("availability slot does not belong to the requested doctor")
	}

	if startTime.Before(time.Now().UTC()) {
		return nil, errors.New("availability slot is in the past")
	}

	if isBooked {
		return nil, repository.ErrSlotAlreadyBooked
	}

	// 5. Check wallet balance and deduct consult fee
	err = s.walletSvc.Deduct(ctx, patientUserID, docProfile.ConsultationFee, fmt.Sprintf("Consultation booking with doctor %s", docProfile.FullName))
	if err != nil {
		if errors.Is(err, wallet.ErrInsufficientBalance) {
			return nil, ErrInsufficientBalance
		}
		return nil, fmt.Errorf("failed to process wallet payment: %w", err)
	}

	// 6. Persist appointment with lock
	apt := &model.Appointment{
		ID:             uuid.New(),
		PatientID:      patUUID,
		DoctorID:       docUUID,
		AvailabilityID: availUUID,
		Status:         "confirmed", // atomically paid
		ScheduledAt:    startTime,
	}

	err = s.repo.CreateWithLock(ctx, apt)
	if err != nil {
		// Rollback wallet charge on failed booking (e.g. concurrent race booking)
		_ = s.walletSvc.Refund(ctx, patientUserID, docProfile.ConsultationFee, "Refund for failed booking transaction")
		if errors.Is(err, repository.ErrSlotAlreadyBooked) {
			return nil, repository.ErrSlotAlreadyBooked
		}
		return nil, err
	}

	if s.consultationSvc != nil {
		if errCons := s.consultationSvc.CreateConsultation(ctx, apt.ID); errCons != nil {
			return nil, fmt.Errorf("failed to create consultation session: %w", errCons)
		}
	}
	if s.notificationSvc != nil {
		// 1. Notify Patient
		_ = s.notificationSvc.PublishNotification(ctx, patientUserID, "email", "appointment_confirmed", map[string]any{
			"appointment_id": apt.ID.String(),
			"role":           "patient",
			"scheduled_at":   apt.ScheduledAt.Format(time.RFC3339),
			"doctor_name":    docProfile.FullName,
		})

		// 2. Notify Doctor
		if docUserID, errParser := uuid.Parse(docProfile.UserID); errParser == nil {
			_ = s.notificationSvc.PublishNotification(ctx, docUserID, "email", "appointment_confirmed", map[string]any{
				"appointment_id": apt.ID.String(),
				"role":           "doctor",
				"scheduled_at":   apt.ScheduledAt.Format(time.RFC3339),
				"patient_name":   patProfile.FullName,
			})
		}
	}

	return s.toResponse(apt), nil
}

func (s *AppointmentServiceImpl) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.AppointmentResponse, error) {
	apt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Authorization check: Patient Owner, Doctor Owner, or Admin
	isAdmin := false
	for _, r := range roles {
		if r == "admin" {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		authorized := false
		for _, r := range roles {
			if r == "patient" {
				patProfile, err := s.patientSvc.GetProfileByID(ctx, apt.PatientID)
				if err == nil && patProfile.UserID == userID.String() {
					authorized = true
					break
				}
			}
			if r == "doctor" {
				docProfile, err := s.doctorSvc.GetProfileByID(ctx, apt.DoctorID)
				if err == nil && docProfile.UserID == userID.String() {
					authorized = true
					break
				}
			}
		}
		if !authorized {
			return nil, ErrUnauthorized
		}
	}

	return s.toResponse(apt), nil
}

func (s *AppointmentServiceImpl) List(ctx context.Context, userID uuid.UUID, roles []string, statusFilter string) ([]*dto.AppointmentResponse, error) {
	isAdmin := false
	for _, r := range roles {
		if r == "admin" {
			isAdmin = true
			break
		}
	}

	filter := map[string]any{}
	if statusFilter != "" {
		filter["status"] = statusFilter
	}

	if !isAdmin {
		// Non-admins are restricted to their own appointments
		isPatient := false
		isDoctor := false
		for _, r := range roles {
			if r == "patient" {
				isPatient = true
			}
			if r == "doctor" {
				isDoctor = true
			}
		}

		if isPatient {
			patProfile, err := s.patientSvc.GetProfileByUserID(ctx, userID)
			if err != nil {
				return nil, err
			}
			patUUID, _ := uuid.Parse(patProfile.ID)
			filter["patient_id"] = patUUID
		} else if isDoctor {
			docProfile, err := s.doctorSvc.GetProfileByUserID(ctx, userID)
			if err != nil {
				return nil, err
			}
			docUUID, _ := uuid.Parse(docProfile.ID)
			filter["doctor_id"] = docUUID
		} else {
			return nil, ErrUnauthorized
		}
	}

	list, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	resp := make([]*dto.AppointmentResponse, len(list))
	for i, a := range list {
		resp[i] = s.toResponse(a)
	}

	return resp, nil
}

func (s *AppointmentServiceImpl) Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.CancelAppointmentRequest) error {
	apt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if apt.Status == "cancelled" {
		return ErrAppointmentAlreadyCancelled
	}

	// Validate ownership and retrieve Patient Owner UserID for refund
	patProfile, err := s.patientSvc.GetProfileByID(ctx, apt.PatientID)
	if err != nil {
		return err
	}
	patOwnerUserID, _ := uuid.Parse(patProfile.UserID)

	authorized := false
	// Is it the patient?
	if patProfile.UserID == userID.String() {
		authorized = true
	} else {
		// Is it the doctor?
		docProfile, err := s.doctorSvc.GetProfileByID(ctx, apt.DoctorID)
		if err == nil && docProfile.UserID == userID.String() {
			authorized = true
		}
	}

	if !authorized {
		return ErrUnauthorized
	}

	// Check cancellation cutoff
	now := time.Now().UTC()
	cutoffTime := apt.ScheduledAt.Add(-time.Duration(s.cutoffMinutes) * time.Minute)
	shouldRefund := !now.After(cutoffTime)

	// Fetch doctor profile to get consult fee
	docProfile, err := s.doctorSvc.GetProfileByID(ctx, apt.DoctorID)
	if err != nil {
		return err
	}

	// Perform cancellation and release availability slot in repository
	err = s.repo.UpdateStatusWithLock(ctx, id, "cancelled", &req.CancelReason)
	if err != nil {
		return err
	}

	if s.consultationSvc != nil {
		_ = s.consultationSvc.CancelConsultation(ctx, id)
	}

	// Process refund if cancelled before cutoff window
	if shouldRefund {
		_ = s.walletSvc.Refund(ctx, patOwnerUserID, docProfile.ConsultationFee, "Refund for cancelled appointment (prior to cutoff)")
	}

	return nil
}

func (s *AppointmentServiceImpl) Reschedule(ctx context.Context, id uuid.UUID, userID uuid.UUID, req dto.RescheduleAppointmentRequest) (*dto.AppointmentResponse, error) {
	newAvailUUID, err := validator.ValidateRescheduleAppointment(req)
	if err != nil {
		return nil, err
	}

	apt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if apt.Status == "cancelled" {
		return nil, ErrAppointmentAlreadyCancelled
	}

	// Verify authorization
	patProfile, err := s.patientSvc.GetProfileByID(ctx, apt.PatientID)
	if err != nil {
		return nil, err
	}
	authorized := false
	if patProfile.UserID == userID.String() {
		authorized = true
	} else {
		docProfile, err := s.doctorSvc.GetProfileByID(ctx, apt.DoctorID)
		if err == nil && docProfile.UserID == userID.String() {
			authorized = true
		}
	}
	if !authorized {
		return nil, ErrUnauthorized
	}

	// Check reschedule cutoff (cannot reschedule within cutoff window)
	now := time.Now().UTC()
	cutoffTime := apt.ScheduledAt.Add(-time.Duration(s.cutoffMinutes) * time.Minute)
	if now.After(cutoffTime) {
		return nil, ErrCancellationCutoffExpired
	}

	// Fetch new slot details and validate doctor matching
	isBooked, slotDocID, startTime, _, err := s.repo.GetAvailabilityByID(ctx, newAvailUUID)
	if err != nil {
		if errors.Is(err, repository.ErrAvailabilityNotFound) {
			return nil, repository.ErrAvailabilityNotFound
		}
		return nil, err
	}

	if slotDocID != apt.DoctorID {
		return nil, errors.New("rescheduled slot does not belong to the same doctor")
	}

	if startTime.Before(now) {
		return nil, errors.New("rescheduled slot is in the past")
	}

	if isBooked {
		return nil, repository.ErrSlotAlreadyBooked
	}

	// Execute reschedule with lock (swapping slot is_booked states)
	err = s.repo.RescheduleWithLock(ctx, id, newAvailUUID, startTime)
	if err != nil {
		if errors.Is(err, repository.ErrSlotAlreadyBooked) {
			return nil, repository.ErrSlotAlreadyBooked
		}
		return nil, err
	}

	apt.AvailabilityID = newAvailUUID
	apt.ScheduledAt = startTime
	apt.UpdatedAt = time.Now().UTC()

	return s.toResponse(apt), nil
}

func (s *AppointmentServiceImpl) toResponse(a *model.Appointment) *dto.AppointmentResponse {
	var cancelledAtStr *string
	if a.CancelledAt != nil {
		tStr := a.CancelledAt.Format(time.RFC3339)
		cancelledAtStr = &tStr
	}

	return &dto.AppointmentResponse{
		ID:             a.ID.String(),
		PatientID:      a.PatientID.String(),
		DoctorID:       a.DoctorID.String(),
		AvailabilityID: a.AvailabilityID.String(),
		Status:         a.Status,
		ScheduledAt:    a.ScheduledAt.Format(time.RFC3339),
		CancelledAt:    cancelledAtStr,
		CancelReason:   a.CancelReason,
		CreatedAt:      a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      a.UpdatedAt.Format(time.RFC3339),
	}
}
