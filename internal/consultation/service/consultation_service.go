package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	appointmentService "github.com/timurdianradhasejati/telemed_hub/internal/appointment/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/repository"
)

type ConsultationServiceImpl struct {
	repo           repository.ConsultationRepository
	appointmentSvc appointmentService.AppointmentService
}

func NewConsultationService(
	repo repository.ConsultationRepository,
	appointmentSvc appointmentService.AppointmentService,
) *ConsultationServiceImpl {
	return &ConsultationServiceImpl{
		repo:           repo,
		appointmentSvc: appointmentSvc,
	}
}

func (s *ConsultationServiceImpl) CreateConsultation(ctx context.Context, appointmentID uuid.UUID) error {
	cons := &model.Consultation{
		ID:            uuid.New(),
		AppointmentID: appointmentID,
		Status:        "scheduled",
	}

	err := s.repo.Create(ctx, cons)
	if err != nil {
		return fmt.Errorf("failed to save consultation record: %w", err)
	}

	return nil
}

func (s *ConsultationServiceImpl) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, roles []string) (*dto.ConsultationResponse, error) {
	cons, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify authorization by querying appointment (centralized authorization).
	// The appointment response also provides DoctorID and PatientID for downstream use.
	appt, err := s.appointmentSvc.GetByID(ctx, cons.AppointmentID, userID, roles)
	if err != nil {
		if errors.Is(err, appointmentService.ErrUnauthorized) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	resp := s.toResponse(cons)
	resp.DoctorID = appt.DoctorID
	resp.PatientID = appt.PatientID
	return resp, nil
}

func (s *ConsultationServiceImpl) Start(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID) (*dto.ConsultationResponse, error) {
	cons, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if cons.Status != "scheduled" {
		return nil, fmt.Errorf("%w: current status is %s, expected scheduled", ErrInvalidTransition, cons.Status)
	}

	// Verify the caller is the assigned doctor of the appointment
	_, err = s.appointmentSvc.GetByID(ctx, cons.AppointmentID, doctorUserID, []string{"doctor"})
	if err != nil {
		return nil, ErrUnauthorized
	}

	now := time.Now().UTC()
	cons.Status = "in_progress"
	cons.StartedAt = &now

	err = s.repo.Update(ctx, cons)
	if err != nil {
		return nil, err
	}

	return s.toResponse(cons), nil
}

func (s *ConsultationServiceImpl) Complete(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID) (*dto.ConsultationResponse, error) {
	cons, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if cons.Status != "in_progress" {
		return nil, fmt.Errorf("%w: current status is %s, expected in_progress", ErrInvalidTransition, cons.Status)
	}

	// Verify the caller is the assigned doctor of the appointment
	_, err = s.appointmentSvc.GetByID(ctx, cons.AppointmentID, doctorUserID, []string{"doctor"})
	if err != nil {
		return nil, ErrUnauthorized
	}

	now := time.Now().UTC()
	cons.Status = "completed"
	cons.EndedAt = &now

	err = s.repo.Update(ctx, cons)
	if err != nil {
		return nil, err
	}

	return s.toResponse(cons), nil
}

func (s *ConsultationServiceImpl) UpdateNotes(ctx context.Context, id uuid.UUID, doctorUserID uuid.UUID, notes string) (*dto.ConsultationResponse, error) {
	cons, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if cons.Status != "in_progress" {
		return nil, fmt.Errorf("%w: current status is %s, expected in_progress", ErrInvalidTransition, cons.Status)
	}

	// Verify doctor authorization
	_, err = s.appointmentSvc.GetByID(ctx, cons.AppointmentID, doctorUserID, []string{"doctor"})
	if err != nil {
		return nil, ErrUnauthorized
	}

	cons.Notes = &notes

	err = s.repo.Update(ctx, cons)
	if err != nil {
		return nil, err
	}

	return s.toResponse(cons), nil
}

func (s *ConsultationServiceImpl) CancelConsultation(ctx context.Context, appointmentID uuid.UUID) error {
	cons, err := s.repo.GetByAppointmentID(ctx, appointmentID)
	if err != nil {
		if errors.Is(err, repository.ErrConsultationNotFound) {
			return nil // no consultation to cancel
		}
		return err
	}

	if cons.Status == "cancelled" || cons.Status == "completed" {
		return nil
	}

	cons.Status = "cancelled"
	return s.repo.Update(ctx, cons)
}

func (s *ConsultationServiceImpl) toResponse(c *model.Consultation) *dto.ConsultationResponse {
	var startStr, endStr *string
	if c.StartedAt != nil {
		tStr := c.StartedAt.Format(time.RFC3339)
		startStr = &tStr
	}
	if c.EndedAt != nil {
		tStr := c.EndedAt.Format(time.RFC3339)
		endStr = &tStr
	}

	return &dto.ConsultationResponse{
		ID:            c.ID.String(),
		AppointmentID: c.AppointmentID.String(),
		Status:        c.Status,
		Notes:         c.Notes,
		StartedAt:     startStr,
		EndedAt:       endStr,
		CreatedAt:     c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     c.UpdatedAt.Format(time.RFC3339),
	}
}
