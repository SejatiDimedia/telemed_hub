package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/mapper"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/validator"
	"github.com/timurdianradhasejati/telemed_hub/internal/shared"
)

type DoctorServiceImpl struct {
	repo         repository.DoctorRepository
	rdb          *redis.Client
	auditService shared.AuditService
}

func NewDoctorService(repo repository.DoctorRepository, rdb *redis.Client, auditService shared.AuditService) *DoctorServiceImpl {
	return &DoctorServiceImpl{
		repo:         repo,
		rdb:          rdb,
		auditService: auditService,
	}
}

func (s *DoctorServiceImpl) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*dto.DoctorResponse, error) {
	doctor, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapper.ToResponse(doctor), nil
}

func (s *DoctorServiceImpl) GetProfileByID(ctx context.Context, id uuid.UUID) (*dto.DoctorResponse, error) {
	doctor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapper.ToResponse(doctor), nil
}

func (s *DoctorServiceImpl) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateDoctorRequest) (*dto.DoctorResponse, error) {
	if err := validator.ValidateUpdateDoctor(req); err != nil {
		return nil, err
	}

	doctor, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	specialtyIDStr := strings.TrimSpace(req.SpecialtyID)
	var specIDPtr *uuid.UUID
	if specialtyIDStr != "" {
		parsedID, err := uuid.Parse(specialtyIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid specialty_id format: %w", err)
		}
		specIDPtr = &parsedID
	}

	license := strings.TrimSpace(req.LicenseNumber)
	phone := strings.TrimSpace(req.PhoneNumber)

	doctor.SpecialtyID = specIDPtr
	doctor.LicenseNumber = &license
	doctor.ConsultationFee = req.ConsultationFee
	doctor.PhoneNumber = &phone

	if err := s.repo.Update(ctx, doctor); err != nil {
		return nil, err
	}

	_ = s.InvalidateDoctorListCache(ctx)

	return mapper.ToResponse(doctor), nil
}

func (s *DoctorServiceImpl) VerifyDoctor(ctx context.Context, adminUserID uuid.UUID, doctorID uuid.UUID, ipAddress string, userAgent string) error {
	// Verify in DB
	if err := s.repo.Verify(ctx, doctorID); err != nil {
		return err
	}

	_ = s.InvalidateDoctorListCache(ctx)

	// Write audit log entry via shared AuditService
	if s.auditService != nil {
		_ = s.auditService.Log(ctx, shared.AuditEntry{
			ActorID:    adminUserID,
			Action:     "doctor.verified",
			TargetType: "doctors",
			TargetID:   doctorID,
			IPAddress:  ipAddress,
			UserAgent:  userAgent,
			Metadata: map[string]any{
				"verified_at":            time.Now().UTC().Format(time.RFC3339),
				"is_credential_verified": true,
			},
		})
	}

	return nil
}

func (s *DoctorServiceImpl) ListDoctors(ctx context.Context, specialty *string, onlyVerified bool, sortBy string, order string, page int, limit int) ([]*dto.DoctorResponse, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}

	offset := (page - 1) * limit

	var specStr string
	if specialty != nil {
		specStr = *specialty
	}

	// Try reading from cache
	cacheKey := fmt.Sprintf("doctor:list:specialty:%s:verified:%t:sort:%s:order:%s:page:%d:limit:%d",
		specStr, onlyVerified, sortBy, order, page, limit)

	if s.rdb != nil {
		cachedBytes, err := s.rdb.Get(ctx, cacheKey).Bytes()
		if err == nil {
			type CachedDoctorList struct {
				Doctors    []*dto.DoctorResponse `json:"doctors"`
				TotalItems int                   `json:"total_items"`
			}
			var cached CachedDoctorList
			if err := json.Unmarshal(cachedBytes, &cached); err == nil {
				return cached.Doctors, cached.TotalItems, nil
			}
		}
	}

	doctors, totalItems, err := s.repo.List(ctx, specialty, onlyVerified, sortBy, order, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	respList := mapper.ToResponseList(doctors)

	// Save to cache
	if s.rdb != nil {
		type CachedDoctorList struct {
			Doctors    []*dto.DoctorResponse `json:"doctors"`
			TotalItems int                   `json:"total_items"`
		}
		cached := CachedDoctorList{
			Doctors:    respList,
			TotalItems: totalItems,
		}
		if cachedBytes, err := json.Marshal(cached); err == nil {
			ttl := 5 * time.Minute
			_ = s.rdb.Set(ctx, cacheKey, cachedBytes, ttl).Err()
			_ = s.rdb.SAdd(ctx, "doctor:list:keys", cacheKey).Err()
			_ = s.rdb.Expire(ctx, "doctor:list:keys", ttl).Err()
		}
	}

	return respList, totalItems, nil
}

func (s *DoctorServiceImpl) AddAvailability(ctx context.Context, doctorUserID uuid.UUID, req dto.CreateAvailabilityRequest) (*dto.AvailabilityResponse, error) {
	// 1. Get Doctor profile
	doctor, err := s.repo.GetByUserID(ctx, doctorUserID)
	if err != nil {
		return nil, err
	}

	// 2. Validate input and parse times
	startTime, endTime, err := validator.ValidateCreateAvailability(req)
	if err != nil {
		return nil, err
	}

	// 3. Check for overlapping slot
	overlapping, err := s.repo.CheckOverlappingSlot(ctx, doctor.ID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	if overlapping {
		return nil, ErrOverlappingSlot
	}

	// 4. Save slot
	slot := &model.Availability{
		DoctorID:  doctor.ID,
		StartTime: startTime,
		EndTime:   endTime,
		IsBooked:  false,
	}

	err = s.repo.CreateAvailability(ctx, slot)
	if err != nil {
		return nil, err
	}

	_ = s.InvalidateAvailabilityCache(ctx, doctor.ID)

	return mapper.ToAvailabilityResponse(slot), nil
}

func (s *DoctorServiceImpl) AddAvailabilityBulk(ctx context.Context, doctorUserID uuid.UUID, req dto.CreateAvailabilityBulkRequest) (*dto.CreateAvailabilityBulkResponse, error) {
	doctor, err := s.repo.GetByUserID(ctx, doctorUserID)
	if err != nil {
		return nil, err
	}

	res := &dto.CreateAvailabilityBulkResponse{
		Created: []*dto.AvailabilityResponse{},
		Errors:  []string{},
	}

	for _, slotReq := range req.Slots {
		startTime, endTime, err := validator.ValidateCreateAvailability(slotReq)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Slot %s-%s validasi gagal: %v", slotReq.StartTime, slotReq.EndTime, err))
			continue
		}

		overlapping, err := s.repo.CheckOverlappingSlot(ctx, doctor.ID, startTime, endTime)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Slot %s-%s db error: %v", slotReq.StartTime, slotReq.EndTime, err))
			continue
		}
		if overlapping {
			res.Errors = append(res.Errors, fmt.Sprintf("Slot %s - %s bentrok dengan jadwal yang sudah ada", slotReq.StartTime, slotReq.EndTime))
			continue
		}

		slot := &model.Availability{
			DoctorID:  doctor.ID,
			StartTime: startTime,
			EndTime:   endTime,
			IsBooked:  false,
		}

		err = s.repo.CreateAvailability(ctx, slot)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Slot %s-%s gagal disimpan: %v", slotReq.StartTime, slotReq.EndTime, err))
			continue
		}

		res.Created = append(res.Created, mapper.ToAvailabilityResponse(slot))
	}

	if len(res.Created) > 0 {
		_ = s.InvalidateAvailabilityCache(ctx, doctor.ID)
	}

	return res, nil
}

func (s *DoctorServiceImpl) RemoveAvailability(ctx context.Context, doctorUserID uuid.UUID, slotID uuid.UUID) error {
	// 1. Get Doctor profile
	doctor, err := s.repo.GetByUserID(ctx, doctorUserID)
	if err != nil {
		return err
	}

	// 2. Perform deletion (repo checks ownership and booking status)
	err = s.repo.DeleteAvailability(ctx, doctor.ID, slotID)
	if err != nil {
		return err
	}

	_ = s.InvalidateAvailabilityCache(ctx, doctor.ID)
	return nil
}

func (s *DoctorServiceImpl) GetAvailability(ctx context.Context, doctorID uuid.UUID, startTimeStr, endTimeStr string, isBooked *bool) ([]*dto.AvailabilityResponse, error) {
	var startTime, endTime time.Time
	var err error

	if startTimeStr == "" {
		startTime = time.Now().UTC()
	} else {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return nil, err
		}
	}

	if endTimeStr == "" {
		endTime = startTime.Add(7 * 24 * time.Hour)
	} else {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, err
		}
	}

	var bookedStr string
	if isBooked != nil {
		bookedStr = fmt.Sprintf("%t", *isBooked)
	} else {
		bookedStr = "nil"
	}

	// Try reading from cache
	cacheKey := fmt.Sprintf("doctor:availability:id:%s:start:%s:end:%s:booked:%s",
		doctorID.String(), startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), bookedStr)

	if s.rdb != nil {
		cachedBytes, err := s.rdb.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var cached []*dto.AvailabilityResponse
			if err := json.Unmarshal(cachedBytes, &cached); err == nil {
				return cached, nil
			}
		}
	}

	slots, err := s.repo.ListAvailability(ctx, doctorID, startTime, endTime, isBooked)
	if err != nil {
		return nil, err
	}

	respList := mapper.ToAvailabilityResponseList(slots)

	// Save to cache
	if s.rdb != nil {
		if cachedBytes, err := json.Marshal(respList); err == nil {
			ttl := 5 * time.Minute
			trackingKey := fmt.Sprintf("doctor:availability:keys:%s", doctorID.String())
			_ = s.rdb.Set(ctx, cacheKey, cachedBytes, ttl).Err()
			_ = s.rdb.SAdd(ctx, trackingKey, cacheKey).Err()
			_ = s.rdb.Expire(ctx, trackingKey, ttl).Err()
		}
	}

	return respList, nil
}

func (s *DoctorServiceImpl) InvalidateAvailabilityCache(ctx context.Context, doctorID uuid.UUID) error {
	if s.rdb == nil {
		return nil
	}

	trackingKey := fmt.Sprintf("doctor:availability:keys:%s", doctorID.String())
	keys, err := s.rdb.SMembers(ctx, trackingKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get availability cache keys from set: %w", err)
	}

	if len(keys) > 0 {
		keys = append(keys, trackingKey)
		err = s.rdb.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete availability cache keys: %w", err)
		}
	} else {
		_ = s.rdb.Del(ctx, trackingKey).Err()
	}

	return nil
}

func (s *DoctorServiceImpl) InvalidateDoctorListCache(ctx context.Context) error {
	if s.rdb == nil {
		return nil
	}

	keys, err := s.rdb.SMembers(ctx, "doctor:list:keys").Result()
	if err != nil {
		return fmt.Errorf("failed to get doctor list cache keys: %w", err)
	}

	if len(keys) > 0 {
		keys = append(keys, "doctor:list:keys")
		err = s.rdb.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete doctor list cache: %w", err)
		}
	} else {
		_ = s.rdb.Del(ctx, "doctor:list:keys").Err()
	}

	return nil
}
