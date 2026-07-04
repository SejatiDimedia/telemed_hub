package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/validator"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type DoctorHandler struct {
	svc    service.DoctorService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewDoctorHandler(svc service.DoctorService, cfg *config.Config, rdb *redis.Client, logger *slog.Logger) *DoctorHandler {
	return &DoctorHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

// Routes wires the Doctor HTTP routes under /api/v1/doctors.
func (h *DoctorHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public lists and detail (optional auth to identify admin role)
	r.Group(func(r chi.Router) {
		r.Use(middleware.OptionalAuthMiddleware(h.cfg, h.rdb))
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
	})

	// Private Doctor actions
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))
		r.Use(middleware.RequireRole("doctor"))
		r.Get("/me", h.GetMe)
		r.Put("/me", h.UpdateMe)
		r.Post("/me/availability", h.AddAvailabilitySlot)
		r.Delete("/me/availability/{slotId}", h.RemoveAvailabilitySlot)
	})

	// Authenticated actions (Any role)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))
		r.Get("/{id}/availability", h.GetAvailabilitySlots)
	})

	// Admin actions
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))
		r.Use(middleware.RequireRole("admin"))
		r.Post("/{id}/verify", h.Verify)
	})

	return r
}

// GetMe handler.
// GET /doctors/me
func (h *DoctorHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	resp, err := h.svc.GetProfileByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrDoctorNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "Doctor profile not found")
			return
		}
		h.logger.Error("failed to fetch doctor profile", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

// UpdateMe handler.
// PUT /doctors/me
func (h *DoctorHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	var req dto.UpdateDoctorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	resp, err := h.svc.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			details := validator.ExtractValidationDetails(err)
			httpresponse.JSON(w, http.StatusBadRequest, httpresponse.Response{
				Success:   false,
				Error:     "Validation failed",
				ErrorCode: "VALIDATION_ERROR",
				Data:      details,
			})
			return
		}
		if errors.Is(err, repository.ErrDoctorNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "Doctor profile not found")
			return
		}
		h.logger.Error("failed to update doctor profile", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

// GetByID handler.
// GET /doctors/{id}
func (h *DoctorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	doctorID, err := uuid.Parse(idParam)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid doctor ID format")
		return
	}

	resp, err := h.svc.GetProfileByID(r.Context(), doctorID)
	if err != nil {
		if errors.Is(err, repository.ErrDoctorNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "Doctor profile not found")
			return
		}
		h.logger.Error("failed to fetch doctor by ID", "error", err)
		httpresponse.InternalError(w)
		return
	}

	// Security rules:
	// Only the doctor themselves, or an admin, can see sensitive credentials (phone_number, license_number).
	// We check if current context has credentials matching user ID, or role is admin.
	loggedInUserID, err := middleware.GetUserID(r.Context())
	isAdmin := false
	if err == nil {
		roles, _ := middleware.GetUserRoles(r.Context())
		for _, r := range roles {
			if r == "admin" {
				isAdmin = true
			}
		}
	}

	isSelf := err == nil && resp.UserID == loggedInUserID.String()

	if !isSelf && !isAdmin {
		sanitized := resp.SanitizeForPublic()
		httpresponse.Success(w, &sanitized)
		return
	}

	httpresponse.Success(w, resp)
}

// Verify handler.
// POST /doctors/{id}/verify
func (h *DoctorHandler) Verify(w http.ResponseWriter, r *http.Request) {
	adminUserID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	idParam := chi.URLParam(r, "id")
	doctorID, err := uuid.Parse(idParam)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid doctor ID format")
		return
	}

	ipAddress := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	err = h.svc.VerifyDoctor(r.Context(), adminUserID, doctorID, ipAddress, userAgent)
	if err != nil {
		if errors.Is(err, repository.ErrDoctorNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "Doctor profile not found")
			return
		}
		h.logger.Error("failed to verify doctor", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.SuccessWithMessage(w, "doctor profile verified successfully", nil)
}

// List handler.
// GET /doctors
func (h *DoctorHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	specialty := q.Get("specialty")
	sortBy := q.Get("sort_by")
	if sortBy == "" {
		sortBy = "created_at"
	}
	order := q.Get("order")
	if order == "" {
		order = "desc"
	}

	page := 1
	if pStr := q.Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 10
	if lStr := q.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Authorization filter:
	// Admin can see unverified doctors; all others (anonymous, patients, doctors) only see verified doctors.
	onlyVerified := true
	if roles, err := middleware.GetUserRoles(r.Context()); err == nil {
		for _, r := range roles {
			if r == "admin" {
				onlyVerified = false
			}
		}
	}

	var specialtyPtr *string
	if specialty != "" {
		specialtyPtr = &specialty
	}

	doctors, totalItems, err := h.svc.ListDoctors(r.Context(), specialtyPtr, onlyVerified, sortBy, order, page, limit)
	if err != nil {
		h.logger.Error("failed to query doctors list", "error", err)
		httpresponse.InternalError(w)
		return
	}

	totalPages := (totalItems + limit - 1) / limit

	// Format paginated response
	httpresponse.JSON(w, http.StatusOK, httpresponse.Response{
		Success:   true,
		Data:      doctors,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Pagination: &httpresponse.PaginationInfo{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	})
}

// POST /doctors/me/availability
func (h *DoctorHandler) AddAvailabilitySlot(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	var req dto.CreateAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed request JSON")
		return
	}

	resp, err := h.svc.AddAvailability(r.Context(), userID, req)
	if err != nil {
		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			httpresponse.JSON(w, http.StatusBadRequest, httpresponse.Response{
				Success:   false,
				Error:     "validation failed",
				ErrorCode: "VALIDATION_FAILED",
				Data:      validator.ExtractValidationDetails(err),
			})
			return
		}
		if errors.Is(err, service.ErrOverlappingSlot) {
			httpresponse.Error(w, http.StatusConflict, "SLOT_OVERLAP", err.Error())
			return
		}
		if errors.Is(err, repository.ErrDoctorNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "DOCTOR_NOT_FOUND", err.Error())
			return
		}

		h.logger.Error("failed to create availability slot", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.JSON(w, http.StatusCreated, httpresponse.Response{
		Success: true,
		Data:    resp,
	})
}

// DELETE /doctors/me/availability/{slotId}
func (h *DoctorHandler) RemoveAvailabilitySlot(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	slotIDStr := chi.URLParam(r, "slotId")
	slotID, err := uuid.Parse(slotIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_SLOT_ID", "invalid slot UUID format")
		return
	}

	err = h.svc.RemoveAvailability(r.Context(), userID, slotID)
	if err != nil {
		if errors.Is(err, repository.ErrAvailabilityNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "SLOT_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, repository.ErrSlotBooked) {
			httpresponse.Error(w, http.StatusBadRequest, "SLOT_BOOKED", err.Error())
			return
		}
		if errors.Is(err, repository.ErrDoctorNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "DOCTOR_NOT_FOUND", err.Error())
			return
		}

		h.logger.Error("failed to delete availability slot", "error", err)
		httpresponse.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /doctors/{id}/availability
func (h *DoctorHandler) GetAvailabilitySlots(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	doctorID, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_DOCTOR_ID", "invalid doctor UUID format")
		return
	}

	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	var isBookedPtr *bool
	if ibStr := r.URL.Query().Get("is_booked"); ibStr != "" {
		if ib, err := strconv.ParseBool(ibStr); err == nil {
			isBookedPtr = &ib
		}
	}

	slots, err := h.svc.GetAvailability(r.Context(), doctorID, startTimeStr, endTimeStr, isBookedPtr)
	if err != nil {
		h.logger.Error("failed to fetch availability slots", "error", err)
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_QUERY_PARAMETERS", "failed to parse query filters: "+err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, httpresponse.Response{
		Success: true,
		Data:    slots,
	})
}

// Helpers
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
