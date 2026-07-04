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
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/validator"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

// PrescriptionHandler handles HTTP requests for the prescription module.
type PrescriptionHandler struct {
	svc    service.PrescriptionService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewPrescriptionHandler(
	svc service.PrescriptionService,
	cfg *config.Config,
	rdb *redis.Client,
	logger *slog.Logger,
) *PrescriptionHandler {
	return &PrescriptionHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

// Routes mounts all prescription endpoints under the given router prefix.
func (h *PrescriptionHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))

	r.Post("/", h.Issue)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)

	return r
}

// Issue handles POST /prescriptions — doctor only.
func (h *PrescriptionHandler) Issue(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	// Only doctors can issue prescriptions
	if !hasRole(roles, "doctor") {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "only doctors can issue prescriptions")
		return
	}

	var req dto.CreatePrescriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "malformed JSON request body")
		return
	}

	// Validate input fields
	if _, err := validator.ValidateCreatePrescription(req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.svc.Issue(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidConsultationStatus):
			httpresponse.Error(w, http.StatusUnprocessableEntity, "INVALID_CONSULTATION_STATUS",
				"prescription can only be issued for in_progress or completed consultations")
		case errors.Is(err, service.ErrUnauthorized):
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN",
				"only the assigned doctor can issue a prescription for this consultation")
		default:
			h.logger.Error("failed to issue prescription", slog.String("error", err.Error()))
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"failed to issue prescription")
		}
		return
	}

	httpresponse.Created(w, resp)
}

// GetByID handles GET /prescriptions/{id}.
func (h *PrescriptionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_PRESCRIPTION_ID", "invalid prescription UUID format")
		return
	}

	resp, err := h.svc.GetByID(r.Context(), id, userID, roles)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnauthorized):
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized access to prescription")
		case errors.Is(err, service.ErrPrescriptionNotFound):
			httpresponse.Error(w, http.StatusNotFound, "PRESCRIPTION_NOT_FOUND", "prescription not found")
		default:
			h.logger.Error("failed to get prescription", slog.String("error", err.Error()))
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get prescription")
		}
		return
	}

	httpresponse.Success(w, resp)
}

// List handles GET /prescriptions — returns prescriptions scoped to caller's role.
func (h *PrescriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	records, err := h.svc.List(r.Context(), userID, roles)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized")
			return
		}
		h.logger.Error("failed to list prescriptions", slog.String("error", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list prescriptions")
		return
	}

	httpresponse.Success(w, records)
}

// hasRole checks whether a given role exists in the roles slice.
func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}
