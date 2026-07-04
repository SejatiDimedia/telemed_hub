package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/model"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateWithLock(ctx context.Context, apt *model.Appointment) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Lock doctor availability row for update to prevent race conditions
	queryLockAvailability := `
		SELECT is_booked 
		FROM doctor_availability 
		WHERE id = $1 
		FOR UPDATE`

	var isBooked bool
	err = tx.QueryRow(ctx, queryLockAvailability, apt.AvailabilityID).Scan(&isBooked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAvailabilityNotFound
		}
		return fmt.Errorf("failed to lock availability row: %w", err)
	}

	if isBooked {
		return ErrSlotAlreadyBooked
	}

	// 2. Mark doctor availability slot as booked
	queryMarkBooked := `
		UPDATE doctor_availability 
		SET is_booked = true, updated_at = NOW() 
		WHERE id = $1`

	_, err = tx.Exec(ctx, queryMarkBooked, apt.AvailabilityID)
	if err != nil {
		return fmt.Errorf("failed to update availability slot: %w", err)
	}

	// 3. Create appointment
	if apt.ID == uuid.Nil {
		apt.ID = uuid.New()
	}
	apt.CreatedAt = time.Now().UTC()
	apt.UpdatedAt = apt.CreatedAt

	queryInsertAppointment := `
		INSERT INTO appointments (id, patient_id, doctor_id, availability_id, status, scheduled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = tx.Exec(ctx, queryInsertAppointment, apt.ID, apt.PatientID, apt.DoctorID, apt.AvailabilityID, apt.Status, apt.ScheduledAt, apt.CreatedAt, apt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert appointment: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Appointment, error) {
	query := `
		SELECT id, patient_id, doctor_id, availability_id, status, scheduled_at, cancelled_at, cancel_reason, created_at, updated_at
		FROM appointments
		WHERE id = $1`

	var a model.Appointment
	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.PatientID, &a.DoctorID, &a.AvailabilityID, &a.Status, &a.ScheduledAt, &a.CancelledAt, &a.CancelReason, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppointmentNotFound
		}
		return nil, fmt.Errorf("failed to fetch appointment: %w", err)
	}

	return &a, nil
}

func (r *PostgresRepository) List(ctx context.Context, filter map[string]any) ([]*model.Appointment, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT id, patient_id, doctor_id, availability_id, status, scheduled_at, cancelled_at, cancel_reason, created_at, updated_at
		FROM appointments
		WHERE 1=1`)

	args := []any{}
	placeholderIdx := 1

	if patientID, ok := filter["patient_id"]; ok {
		queryBuilder.WriteString(fmt.Sprintf(" AND patient_id = $%d", placeholderIdx))
		args = append(args, patientID)
		placeholderIdx++
	}

	if doctorID, ok := filter["doctor_id"]; ok {
		queryBuilder.WriteString(fmt.Sprintf(" AND doctor_id = $%d", placeholderIdx))
		args = append(args, doctorID)
		placeholderIdx++
	}

	if status, ok := filter["status"]; ok {
		queryBuilder.WriteString(fmt.Sprintf(" AND status = $%d", placeholderIdx))
		args = append(args, status)
		placeholderIdx++
	}

	queryBuilder.WriteString(" ORDER BY scheduled_at DESC")

	rows, err := r.db.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query appointments: %w", err)
	}
	defer rows.Close()

	appointments := []*model.Appointment{}
	for rows.Next() {
		var a model.Appointment
		err = rows.Scan(
			&a.ID, &a.PatientID, &a.DoctorID, &a.AvailabilityID, &a.Status, &a.ScheduledAt, &a.CancelledAt, &a.CancelReason, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %w", err)
		}
		appointments = append(appointments, &a)
	}

	return appointments, nil
}

func (r *PostgresRepository) UpdateStatusWithLock(ctx context.Context, id uuid.UUID, status string, cancelReason *string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Lock appointment row and retrieve details
	queryLockApt := `
		SELECT availability_id, status 
		FROM appointments 
		WHERE id = $1 
		FOR UPDATE`

	var oldAvailabilityID uuid.UUID
	var oldStatus string
	err = tx.QueryRow(ctx, queryLockApt, id).Scan(&oldAvailabilityID, &oldStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAppointmentNotFound
		}
		return fmt.Errorf("failed to lock appointment row: %w", err)
	}

	// 2. If status changes to 'cancelled', release availability slot
	if status == "cancelled" && oldStatus != "cancelled" {
		queryReleaseSlot := `
			UPDATE doctor_availability 
			SET is_booked = false, updated_at = NOW() 
			WHERE id = $1`

		_, err = tx.Exec(ctx, queryReleaseSlot, oldAvailabilityID)
		if err != nil {
			return fmt.Errorf("failed to release doctor availability slot: %w", err)
		}
	}

	// 3. Update appointment record status
	now := time.Now().UTC()
	var cancelledAt *time.Time
	if status == "cancelled" {
		cancelledAt = &now
	}

	queryUpdateApt := `
		UPDATE appointments 
		SET status = $1, cancelled_at = $2, cancel_reason = $3, updated_at = NOW() 
		WHERE id = $4`

	_, err = tx.Exec(ctx, queryUpdateApt, status, cancelledAt, cancelReason, id)
	if err != nil {
		return fmt.Errorf("failed to update appointment: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) RescheduleWithLock(ctx context.Context, id uuid.UUID, newAvailabilityID uuid.UUID, scheduledAt time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Lock appointment row and retrieve old availability ID
	queryLockApt := `
		SELECT availability_id 
		FROM appointments 
		WHERE id = $1 
		FOR UPDATE`

	var oldAvailabilityID uuid.UUID
	err = tx.QueryRow(ctx, queryLockApt, id).Scan(&oldAvailabilityID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAppointmentNotFound
		}
		return fmt.Errorf("failed to lock appointment row: %w", err)
	}

	// 2. Release old availability slot
	queryReleaseOld := `
		UPDATE doctor_availability 
		SET is_booked = false, updated_at = NOW() 
		WHERE id = $1`

	_, err = tx.Exec(ctx, queryReleaseOld, oldAvailabilityID)
	if err != nil {
		return fmt.Errorf("failed to release old availability slot: %w", err)
	}

	// 3. Lock new availability slot
	queryLockNew := `
		SELECT is_booked 
		FROM doctor_availability 
		WHERE id = $1 
		FOR UPDATE`

	var isBooked bool
	err = tx.QueryRow(ctx, queryLockNew, newAvailabilityID).Scan(&isBooked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAvailabilityNotFound
		}
		return fmt.Errorf("failed to lock new availability slot: %w", err)
	}

	if isBooked {
		return ErrSlotAlreadyBooked
	}

	// 4. Mark new availability slot as booked
	queryBookNew := `
		UPDATE doctor_availability 
		SET is_booked = true, updated_at = NOW() 
		WHERE id = $1`

	_, err = tx.Exec(ctx, queryBookNew, newAvailabilityID)
	if err != nil {
		return fmt.Errorf("failed to book new availability slot: %w", err)
	}

	// 5. Update appointment with new availability slot and scheduled time
	queryUpdateApt := `
		UPDATE appointments 
		SET availability_id = $1, scheduled_at = $2, updated_at = NOW() 
		WHERE id = $3`

	_, err = tx.Exec(ctx, queryUpdateApt, newAvailabilityID, scheduledAt, id)
	if err != nil {
		return fmt.Errorf("failed to update rescheduled appointment: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetAvailabilityByID(ctx context.Context, id uuid.UUID) (bool, uuid.UUID, time.Time, time.Time, error) {
	query := `
		SELECT is_booked, doctor_id, start_time, end_time 
		FROM doctor_availability 
		WHERE id = $1`

	var isBooked bool
	var doctorID uuid.UUID
	var startTime, endTime time.Time

	err := r.db.QueryRow(ctx, query, id).Scan(&isBooked, &doctorID, &startTime, &endTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, uuid.Nil, time.Time{}, time.Time{}, ErrAvailabilityNotFound
		}
		return false, uuid.Nil, time.Time{}, time.Time{}, fmt.Errorf("failed to query availability slot: %w", err)
	}

	return isBooked, doctorID, startTime, endTime, nil
}
