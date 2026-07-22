package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	ctx := context.Background()

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	require.NoError(t, err)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: endpoint,
	})

	cleanup := func() {
		rdb.Close()
		_ = redisContainer.Terminate(ctx)
	}

	return rdb, cleanup
}

func TestDoctorService_Caching(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	mockRepo := new(MockDoctorRepository)
	svc := NewDoctorService(mockRepo, rdb, nil)

	ctx := context.Background()
	docID := uuid.New()
	doctorUserID := uuid.New()
	phone := "+62811223344"
	specialty := "Cardiology"
	license := "123.456"
	specID := uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")

	docModel := &model.Doctor{
		ID:                   docID,
		UserID:               doctorUserID,
		FullName:             "Dr. Cache-Aside",
		PhoneNumber:          &phone,
		SpecialtyID:          &specID,
		LicenseNumber:        &license,
		IsCredentialVerified: true,
		ConsultationFee:      150000,
	}

	t.Run("ListDoctors caching workflow", func(t *testing.T) {
		// Mock repo to return doctor list
		mockRepo.On("List", mock.Anything, &specialty, true, "name", "asc", 0, 10).Return([]*model.Doctor{docModel}, 1, nil).Once()

		// 1. Cache Miss: call ListDoctors, should fetch from repo and cache in Redis
		docs, total, err := svc.ListDoctors(ctx, &specialty, true, "name", "asc", 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, docs, 1)

		// 2. Cache Hit: call again. Since Mock Repo was configured with Once(),
		// if it hits database again, the test will panic or fail because no mock is registered.
		// Thus, if it returns successfully, it PROVES it came from Redis!
		docs2, total2, err := svc.ListDoctors(ctx, &specialty, true, "name", "asc", 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 1, total2)
		assert.Len(t, docs2, 1)
		assert.Equal(t, docs[0].ID, docs2[0].ID)

		// 3. Cache Invalidation: call UpdateProfile (triggers list cache invalidation)
		mockRepo.On("GetByUserID", mock.Anything, doctorUserID).Return(docModel, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Once()
		updateReq := dto.UpdateDoctorRequest{
			SpecialtyID:     "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			LicenseNumber:   license,
			ConsultationFee: 150000,
			PhoneNumber:     phone,
		}
		_, err = svc.UpdateProfile(ctx, doctorUserID, updateReq)
		require.NoError(t, err)

		// 4. Verify cache is gone (ListDoctors should fail if it tries to fetch from repo because mock is not configured)
		// We register mock once more to let it succeed and re-cache
		mockRepo.On("List", mock.Anything, &specialty, true, "name", "asc", 0, 10).Return([]*model.Doctor{docModel}, 1, nil).Once()
		_, _, err = svc.ListDoctors(ctx, &specialty, true, "name", "asc", 1, 10)
		require.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("GetAvailability caching workflow", func(t *testing.T) {
		slot := &model.Availability{
			ID:        uuid.New(),
			DoctorID:  docID,
			StartTime: time.Now().Add(24 * time.Hour),
			EndTime:   time.Now().Add(25 * time.Hour),
			IsBooked:  false,
		}

		mockRepo.On("ListAvailability", mock.Anything, docID, mock.Anything, mock.Anything, mock.Anything).Return([]*model.Availability{slot}, nil).Once()

		// 1. Cache Miss: call GetAvailability, should fetch from repo and cache
		slots, err := svc.GetAvailability(ctx, docID, "", "", nil)
		require.NoError(t, err)
		assert.Len(t, slots, 1)

		// 2. Cache Hit: call again. Should hit Redis.
		slots2, err := svc.GetAvailability(ctx, docID, "", "", nil)
		require.NoError(t, err)
		assert.Len(t, slots2, 1)

		// 3. Invalidate via direct service call
		err = svc.InvalidateAvailabilityCache(ctx, docID)
		require.NoError(t, err)

		// 4. Verify cache is gone (ListAvailability should fail if it attempts query unless registered again)
		mockRepo.On("ListAvailability", mock.Anything, docID, mock.Anything, mock.Anything, mock.Anything).Return([]*model.Availability{slot}, nil).Once()
		_, err = svc.GetAvailability(ctx, docID, "", "", nil)
		require.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})
}
