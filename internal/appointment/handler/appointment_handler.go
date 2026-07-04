package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/validator"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type AppointmentHandler struct {
	svc    service.AppointmentService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewAppointmentHandler(svc service.AppointmentService, cfg *config.Config, rdb *redis.Client, logger *slog.Logger) *AppointmentHandler {
	return &AppointmentHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

// Routes wires the HTTP routes for appointments under /api/v1/appointments
func (h *AppointmentHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))

		r.Post("/", h.Book)
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Post("/{id}/cancel", h.Cancel)
		r.Post("/{id}/reschedule", h.Reschedule)
	})

	return r
}

func (h *AppointmentHandler) Book(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	// Gate: only Patient role can book appointments
	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}
	isPatient := false
	for _, r := range roles {
		if r == "patient" {
			isPatient = true
			break
		}
	}
	if !isPatient {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Only patients can book appointments")
		return
	}

	var req dto.CreateAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed request JSON")
		return
	}

	resp, err := h.svc.Book(r.Context(), userID, req)
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
		if errors.Is(err, service.ErrProfileIncomplete) {
			httpcallUnprocessable(w, "PROFILE_INCOMPLETE", err.Error())
			return
		}
		if errors.Is(err, service.ErrDoctorNotVerified) {
			httpcallUnprocessable(w, "DOCTOR_NOT_VERIFIED", err.Error())
			return
		}
		if errors.Is(err, service.ErrInsufficientBalance) {
			httpcallUnprocessable(w, "INSUFFICIENT_BALANCE", err.Error())
			return
		}
		if errors.Is(err, repository.ErrSlotAlreadyBooked) {
			httpresponse.Error(w, http.StatusConflict, "SLOT_ALREADY_BOOKED", "this availability slot is already taken")
			return
		}
		if errors.Is(err, repository.ErrAvailabilityNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "SLOT_NOT_FOUND", "availability slot not found")
			return
		}

		h.logger.Error("failed to book appointment", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.JSON(w, http.StatusCreated, httpresponse.Response{
		Success: true,
		Data:    resp,
	})
}

func (h *AppointmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_APPOINTMENT_ID", "invalid appointment UUID format")
		return
	}

	resp, err := h.svc.GetByID(r.Context(), id, userID, roles)
	if err != nil {
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "APPOINTMENT_NOT_FOUND", "appointment not found")
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized access to appointment")
			return
		}

		h.logger.Error("failed to retrieve appointment", "error", err)
		httpcallInternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

func (h *AppointmentHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	statusFilter := r.URL.Query().Get("status")

	resp, err := h.svc.List(r.Context(), userID, roles, statusFilter)
	if err != nil {
		h.logger.Error("failed to query appointments list", "error", err)
		httpcallInternalError(w)
		return
	}

	httpcallSuccess(w, resp)
}

func (h *AppointmentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_APPOINTMENT_ID", "invalid appointment UUID format")
		return
	}

	var req dto.CancelAppointmentRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // ignore error, cancel reason is optional

	err = h.svc.Cancel(r.Context(), id, userID, req)
	if err != nil {
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "APPOINTMENT_NOT_FOUND", "appointment not found")
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized cancellation attempt")
			return
		}
		if errors.Is(err, service.ErrCancellationCutoffExpired) {
			httpcallUnprocessable(w, "CANCELLATION_WINDOW_EXPIRED", "cancellation window expired (cutoff limits reached)")
			return
		}
		if errors.Is(err, service.ErrAppointmentAlreadyCancelled) {
			httpcallUnprocessable(w, "ALREADY_CANCELLED", "appointment is already cancelled")
			return
		}

		h.logger.Error("failed to cancel appointment", "error", err)
		httpcallInternalError(w)
		return
	}

	httpresponse.SuccessWithMessage(w, "appointment cancelled successfully", nil)
}

func (h *AppointmentHandler) Reschedule(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	var req dto.RescheduleAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed request JSON")
		return
	}

	resp, err := h.svc.Reschedule(r.Context(), id, userID, req)
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
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "APPOINTMENT_NOT_FOUND", "appointment not found")
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			httpcallForbidden(w)
			return
		}
		if errors.Is(err, service.ErrCancellationCutoffExpired) {
			httpcallUnprocessable(w, "CANCELLATION_WINDOW_EXPIRED", "cancellation window expired (cutoff limits reached)")
			return
		}
		if errors.Is(err, service.ErrDoctorNotVerified) {
			httpcallUnprocessable(w, "DOCTOR_NOT_VERIFIED", "doctor is not verified")
			return
		}
		if errors.Is(err, repository.ErrSlotAlreadyBooked) {
			httpresponse.Error(w, http.StatusConflict, "SLOT_ALREADY_BOOKED", "this availability slot is already taken")
			return
		}
		if errors.Is(err, repository.ErrAvailabilityNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "SLOT_NOT_FOUND", "availability slot not found")
			return
		}

		h.logger.Error("failed to reschedule appointment", "error", err)
		httpcallInternalError(w)
		return
	}

	httpresponse.SuccessWithMessage(w, "appointment rescheduled successfully", resp)
}

// Helpers/Stubs for clean returns
func httpcallUnauthorized(w http.ResponseWriter) {
	httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
}

func httpcallUnprocessable(w http.ResponseWriter, code, msg string) {
	httpresponse.Error(w, http.StatusUnprocessableEntity, code, msg)
}

func httpcallForbidden(w http.ResponseWriter) {
	httpcallUnauth := http.StatusForbidden
	httpresponse.Error(w, httpcallUnauth, "FORBIDDEN", "Unauthorized action")
}

func httpcallInternalError(w http.ResponseWriter) {
	httpresponse.InternalError(w)
}

func httpcallSuccess(w http.ResponseWriter, data any) {
	httpresponse.Success(w, data)
}
