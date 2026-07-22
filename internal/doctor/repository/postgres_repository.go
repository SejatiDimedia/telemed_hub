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

	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Doctor, error) {
	query := `
		SELECT d.id, d.user_id, u.email, u.full_name, u.phone_number, d.specialty_id, s.id, s.name, s.image_icon, s.description, d.license_number, d.is_credential_verified, d.consultation_fee, d.created_at, d.updated_at, d.deleted_at
		FROM doctors d
		JOIN users u ON d.user_id = u.id
		LEFT JOIN specialties s ON d.specialty_id = s.id
		WHERE d.user_id = $1 AND d.deleted_at IS NULL AND u.deleted_at IS NULL`

	var d model.Doctor
	var sID *uuid.UUID
	var sName, sIcon, sDesc *string

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&d.ID, &d.UserID, &d.Email, &d.FullName, &d.PhoneNumber, &d.SpecialtyID, &sID, &sName, &sIcon, &sDesc, &d.LicenseNumber, &d.IsCredentialVerified, &d.ConsultationFee, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDoctorNotFound
		}
		return nil, fmt.Errorf("failed to fetch doctor by user ID: %w", err)
	}

	if sID != nil {
		d.Specialty = &model.Specialty{
			ID:          *sID,
			Name:        *sName,
			ImageIcon:   *sIcon,
			Description: sDesc,
		}
	}

	return &d, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Doctor, error) {
	query := `
		SELECT d.id, d.user_id, u.email, u.full_name, u.phone_number, d.specialty_id, s.id, s.name, s.image_icon, s.description, d.license_number, d.is_credential_verified, d.consultation_fee, d.created_at, d.updated_at, d.deleted_at
		FROM doctors d
		JOIN users u ON d.user_id = u.id
		LEFT JOIN specialties s ON d.specialty_id = s.id
		WHERE d.id = $1 AND d.deleted_at IS NULL AND u.deleted_at IS NULL`

	var d model.Doctor
	var sID *uuid.UUID
	var sName, sIcon, sDesc *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.UserID, &d.Email, &d.FullName, &d.PhoneNumber, &d.SpecialtyID, &sID, &sName, &sIcon, &sDesc, &d.LicenseNumber, &d.IsCredentialVerified, &d.ConsultationFee, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDoctorNotFound
		}
		return nil, fmt.Errorf("failed to fetch doctor by ID: %w", err)
	}

	if sID != nil {
		d.Specialty = &model.Specialty{
			ID:          *sID,
			Name:        *sName,
			ImageIcon:   *sIcon,
			Description: sDesc,
		}
	}

	return &d, nil
}

func (r *PostgresRepository) Update(ctx context.Context, d *model.Doctor) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	now := time.Now().UTC()

	// 1. Update shared user fields
	queryUser := `
		UPDATE users
		SET phone_number = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL`

	res, err := tx.Exec(ctx, queryUser, d.PhoneNumber, now, d.UserID)
	if err != nil {
		return fmt.Errorf("failed to update user phone: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrDoctorNotFound
	}

	// 2. Update doctor specific fields
	queryDoctor := `
		UPDATE doctors
		SET specialty_id = $1, license_number = $2, consultation_fee = $3, updated_at = $4
		WHERE id = $5 AND deleted_at IS NULL`

	res, err = tx.Exec(ctx, queryDoctor, d.SpecialtyID, d.LicenseNumber, d.ConsultationFee, now, d.ID)
	if err != nil {
		return fmt.Errorf("failed to update doctor fields: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrDoctorNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	d.UpdatedAt = now
	return nil
}

func (r *PostgresRepository) Verify(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE doctors
		SET is_credential_verified = true, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL`

	res, err := r.db.Exec(ctx, query, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to verify doctor credentials: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrDoctorNotFound
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, specialty *string, onlyVerified bool, sortBy string, order string, offset int, limit int) ([]*model.Doctor, int, error) {
	// Prevent SQL injection by validating sorting arguments
	allowedSortColumns := map[string]bool{
		"consultation_fee": true,
		"created_at":       true,
	}
	if !allowedSortColumns[sortBy] {
		sortBy = "created_at"
	}

	order = strings.ToUpper(order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	// Base query construction
	whereClauses := []string{"d.deleted_at IS NULL", "u.deleted_at IS NULL"}
	args := []any{}
	argCount := 1

	if specialty != nil && *specialty != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("s.name ILIKE $%d OR CAST(d.specialty_id AS VARCHAR) = $%d", argCount, argCount))
		args = append(args, "%"+*specialty+"%")
		argCount++
	}

	if onlyVerified {
		whereClauses = append(whereClauses, "d.is_credential_verified = true")
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(d.id)
		FROM doctors d
		JOIN users u ON d.user_id = u.id
		LEFT JOIN specialties s ON d.specialty_id = s.id
		%s`, whereSQL)

	var totalItems int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count doctors list: %w", err)
	}

	// 2. Fetch records
	selectQuery := fmt.Sprintf(`
		SELECT d.id, d.user_id, u.email, u.full_name, u.phone_number, d.specialty_id, s.id, s.name, s.image_icon, s.description, d.license_number, d.is_credential_verified, d.consultation_fee, d.created_at, d.updated_at, d.deleted_at
		FROM doctors d
		JOIN users u ON d.user_id = u.id
		LEFT JOIN specialties s ON d.specialty_id = s.id
		%s
		ORDER BY d.%s %s
		LIMIT $%d OFFSET $%d`, whereSQL, sortBy, order, argCount, argCount+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query doctors list: %w", err)
	}
	defer rows.Close()

	doctors := []*model.Doctor{}
	for rows.Next() {
		var d model.Doctor
		var sID *uuid.UUID
		var sName, sIcon, sDesc *string
		err = rows.Scan(
			&d.ID, &d.UserID, &d.Email, &d.FullName, &d.PhoneNumber, &d.SpecialtyID, &sID, &sName, &sIcon, &sDesc, &d.LicenseNumber, &d.IsCredentialVerified, &d.ConsultationFee, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan doctor row: %w", err)
		}
		if sID != nil {
			d.Specialty = &model.Specialty{
				ID:          *sID,
				Name:        *sName,
				ImageIcon:   *sIcon,
				Description: sDesc,
			}
		}
		doctors = append(doctors, &d)
	}

	return doctors, totalItems, nil
}

func (r *PostgresRepository) CreateAvailability(ctx context.Context, slot *model.Availability) error {
	query := `
		INSERT INTO doctor_availability (id, doctor_id, start_time, end_time, is_booked, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if slot.ID == uuid.Nil {
		slot.ID = uuid.New()
	}
	now := time.Now().UTC()
	slot.CreatedAt = now
	slot.UpdatedAt = now

	_, err := r.db.Exec(ctx, query, slot.ID, slot.DoctorID, slot.StartTime, slot.EndTime, slot.IsBooked, slot.CreatedAt, slot.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert availability slot: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteAvailability(ctx context.Context, doctorID uuid.UUID, slotID uuid.UUID) error {
	// Check if slot exists, belongs to doctor, and is booked
	queryCheck := `SELECT is_booked FROM doctor_availability WHERE id = $1 AND doctor_id = $2`
	var isBooked bool
	err := r.db.QueryRow(ctx, queryCheck, slotID, doctorID).Scan(&isBooked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAvailabilityNotFound
		}
		return fmt.Errorf("failed to check slot status: %w", err)
	}

	if isBooked {
		return ErrSlotBooked
	}

	queryDelete := `DELETE FROM doctor_availability WHERE id = $1 AND doctor_id = $2`
	res, err := r.db.Exec(ctx, queryDelete, slotID, doctorID)
	if err != nil {
		return fmt.Errorf("failed to delete availability slot: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrAvailabilityNotFound
	}
	return nil
}

func (r *PostgresRepository) GetAvailabilityByID(ctx context.Context, slotID uuid.UUID) (*model.Availability, error) {
	query := `
		SELECT id, doctor_id, start_time, end_time, is_booked, created_at, updated_at
		FROM doctor_availability
		WHERE id = $1`
	var s model.Availability
	err := r.db.QueryRow(ctx, query, slotID).Scan(
		&s.ID, &s.DoctorID, &s.StartTime, &s.EndTime, &s.IsBooked, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAvailabilityNotFound
		}
		return nil, fmt.Errorf("failed to fetch availability slot: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) ListAvailability(ctx context.Context, doctorID uuid.UUID, startTime time.Time, endTime time.Time, isBooked *bool) ([]*model.Availability, error) {
	var query strings.Builder
	query.WriteString(`
		SELECT id, doctor_id, start_time, end_time, is_booked, created_at, updated_at
		FROM doctor_availability
		WHERE doctor_id = $1 AND start_time >= $2 AND end_time <= $3`)
	
	args := []any{doctorID, startTime, endTime}
	placeholderIdx := 4

	if isBooked != nil {
		query.WriteString(fmt.Sprintf(" AND is_booked = $%d", placeholderIdx))
		args = append(args, *isBooked)
		placeholderIdx++
	}

	query.WriteString(" ORDER BY start_time ASC")

	rows, err := r.db.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list availability: %w", err)
	}
	defer rows.Close()

	var slots []*model.Availability
	for rows.Next() {
		var s model.Availability
		err := rows.Scan(&s.ID, &s.DoctorID, &s.StartTime, &s.EndTime, &s.IsBooked, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan availability slot: %w", err)
		}
		slots = append(slots, &s)
	}
	return slots, nil
}

func (r *PostgresRepository) CheckOverlappingSlot(ctx context.Context, doctorID uuid.UUID, startTime time.Time, endTime time.Time) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM doctor_availability 
			WHERE doctor_id = $1 
			  AND start_time < $3 
			  AND end_time > $2
		)`
	var exists bool
	err := r.db.QueryRow(ctx, query, doctorID, startTime, endTime).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check overlapping slot: %w", err)
	}
	return exists, nil
}
