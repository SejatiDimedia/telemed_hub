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
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type ConsultationHandler struct {
	svc    service.ConsultationService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewConsultationHandler(svc service.ConsultationService, cfg *config.Config, rdb *redis.Client, logger *slog.Logger) *ConsultationHandler {
	return &ConsultationHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

// Routes returns the router for the consultation module
func (h *ConsultationHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))

		r.Get("/{id}", h.GetByID)
		r.Post("/{id}/start", h.Start)
		r.Post("/{id}/complete", h.Complete)
		r.Put("/{id}/notes", h.UpdateNotes)
	})

	return r
}

func (h *ConsultationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
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
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_CONSULTATION_ID", "invalid consultation UUID format")
		return
	}

	resp, err := h.svc.GetByID(r.Context(), id, userID, roles)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized access to consultation")
			return
		}
		httpresponse.Error(w, http.StatusNotFound, "CONSULTATION_NOT_FOUND", "consultation session not found")
		return
	}

	httpresponse.Success(w, resp)
}

func (h *ConsultationHandler) Start(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	// Only doctor can start consultation
	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}
	isDoctor := false
	for _, role := range roles {
		if role == "doctor" {
			isDoctor = true
			break
		}
	}
	if !isDoctor {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "only doctor can start consultation")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_CONSULTATION_ID", "invalid consultation UUID format")
		return
	}

	resp, err := h.svc.Start(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized doctor access")
			return
		}
		if errors.Is(err, service.ErrInvalidTransition) {
			httpresponse.Error(w, http.StatusUnprocessableEntity, "INVALID_STATE", err.Error())
			return
		}
		h.logger.Error("failed to start consultation", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.SuccessWithMessage(w, "consultation started successfully", resp)
}

func (h *ConsultationHandler) Complete(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.svc.Complete(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized doctor access")
			return
		}
		if errors.Is(err, service.ErrInvalidTransition) {
			httpresponse.Error(w, http.StatusUnprocessableEntity, "INVALID_STATE", err.Error())
			return
		}
		h.logger.Error("failed to complete consultation", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpcallSuccess(w, resp)
}

func (h *ConsultationHandler) UpdateNotes(w http.ResponseWriter, r *http.Request) {
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

	var req dto.UpdateNotesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed request JSON")
		return
	}

	resp, err := h.svc.UpdateNotes(r.Context(), id, userID, req.Notes)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized doctor access")
			return
		}
		if errors.Is(err, service.ErrInvalidTransition) {
			httpresponse.Error(w, http.StatusUnprocessableEntity, "INVALID_STATE", err.Error())
			return
		}
		h.logger.Error("failed to update notes", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpcallSuccess(w, resp)
}

func httpcallUnauthorized(w http.ResponseWriter) {
	httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
}

func httpcallSuccess(w http.ResponseWriter, data any) {
	httpresponse.Success(w, data)
}
