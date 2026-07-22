package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
)

func TestPostgresRepository_AvailabilityOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	// 1. Insert Doctor User
	userID := uuid.New()
	doctorID := uuid.New()

	queryUser := `
		INSERT INTO users (id, email, password_hash, full_name, is_verified, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := db.Exec(ctx, queryUser, userID, "amir.avail@test.com", "hash", "Dr. Amir Avail", true, "active", time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)

	queryDoctor := `
		INSERT INTO doctors (id, user_id, is_credential_verified, consultation_fee, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = db.Exec(ctx, queryDoctor, doctorID, userID, true, 100000, time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)

	// 2. Create Availability Slot
	startTime := time.Now().Add(24 * time.Hour).Truncate(time.Second).UTC()
	endTime := startTime.Add(30 * time.Minute).UTC()

	slot := &model.Availability{
		ID:        uuid.New(),
		DoctorID:  doctorID,
		StartTime: startTime,
		EndTime:   endTime,
		IsBooked:  false,
	}

	err = repo.CreateAvailability(ctx, slot)
	require.NoError(t, err)

	// 3. Get Availability By ID
	fetched, err := repo.GetAvailabilityByID(ctx, slot.ID)
	require.NoError(t, err)
	assert.Equal(t, slot.ID, fetched.ID)
	assert.Equal(t, slot.DoctorID, fetched.DoctorID)
	assert.True(t, slot.StartTime.Equal(fetched.StartTime))
	assert.True(t, slot.EndTime.Equal(fetched.EndTime))
	assert.False(t, fetched.IsBooked)

	// 4. Test Overlap Checks
	// A: Exact overlap
	overlap1, err := repo.CheckOverlappingSlot(ctx, doctorID, startTime, endTime)
	require.NoError(t, err)
	assert.True(t, overlap1)

	// B: Partial overlap (ends inside, starts before)
	overlap2, err := repo.CheckOverlappingSlot(ctx, doctorID, startTime.Add(-10*time.Minute), startTime.Add(10*time.Minute))
	require.NoError(t, err)
	assert.True(t, overlap2)

	// C: Non-overlapping slot
	overlap3, err := repo.CheckOverlappingSlot(ctx, doctorID, endTime, endTime.Add(30*time.Minute))
	require.NoError(t, err)
	assert.False(t, overlap3)

	// 5. List Availability
	// Filter by range containing our slot
	list, err := repo.ListAvailability(ctx, doctorID, startTime.Add(-1*time.Hour), endTime.Add(1*time.Hour), nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// Filter by range excluding our slot
	listEmpty, err := repo.ListAvailability(ctx, doctorID, startTime.Add(-5*time.Hour), startTime.Add(-1*time.Hour), nil)
	require.NoError(t, err)
	assert.Len(t, listEmpty, 0)

	// 6. Delete Booked Slot check
	// Mark slot as booked manually
	_, err = db.Exec(ctx, `UPDATE doctor_availability SET is_booked = true WHERE id = $1`, slot.ID)
	require.NoError(t, err)

	err = repo.DeleteAvailability(ctx, doctorID, slot.ID)
	assert.ErrorIs(t, err, ErrSlotBooked)

	// Unbook and delete
	_, err = db.Exec(ctx, `UPDATE doctor_availability SET is_booked = false WHERE id = $1`, slot.ID)
	require.NoError(t, err)

	err = repo.DeleteAvailability(ctx, doctorID, slot.ID)
	require.NoError(t, err)

	// Verify deleted
	_, err = repo.GetAvailabilityByID(ctx, slot.ID)
	assert.ErrorIs(t, err, ErrAvailabilityNotFound)
}
