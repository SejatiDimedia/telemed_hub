package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/validator"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type PatientHandler struct {
	svc    service.PatientService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewPatientHandler(svc service.PatientService, cfg *config.Config, rdb *redis.Client, logger *slog.Logger) *PatientHandler {
	return &PatientHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

// Routes wires the Patient HTTP routes under /api/v1/patients.
func (h *PatientHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))
		r.Get("/me", h.GetMe)
		r.Put("/me", h.UpdateMe)
		r.Get("/{id}", h.GetByID)
	})

	return r
}

// GetMe handler.
// GET /patients/me
func (h *PatientHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	resp, err := h.svc.GetProfileByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrPatientNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "Patient profile not found")
			return
		}
		h.logger.Error("failed to get patient me profile", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

// UpdateMe handler.
// PUT /patients/me
func (h *PatientHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	var req dto.UpdatePatientRequest
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
		if errors.Is(err, repository.ErrPatientNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "Patient profile not found")
			return
		}
		h.logger.Error("failed to update patient profile", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

// GetByID handler.
// GET /patients/{id}
func (h *PatientHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	loggedInUserID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	idParam := chi.URLParam(r, "id")
	patientID, err := uuid.Parse(idParam)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid patient ID format")
		return
	}

	// Fetch target patient profile
	resp, err := h.svc.GetProfileByID(r.Context(), patientID)
	if err != nil {
		if errors.Is(err, repository.ErrPatientNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "Patient profile not found")
			return
		}
		h.logger.Error("failed to fetch patient by ID", "error", err)
		httpresponse.InternalError(w)
		return
	}

	// Authorization Check:
	// Only the patient themselves, an admin, or a doctor is allowed to access.
	isOwner := resp.UserID == loggedInUserID.String()
	isAdmin := false
	isDoctor := false

	for _, r := range roles {
		if r == "admin" {
			isAdmin = true
		}
		if r == "doctor" {
			isDoctor = true
		}
	}

	if !isOwner && !isAdmin && !isDoctor {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied: insufficient permissions to view this profile")
		return
	}

	httpresponse.Success(w, resp)
}
