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
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/validator"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type MedicalRecordHandler struct {
	svc    service.MedicalRecordService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewMedicalRecordHandler(
	svc service.MedicalRecordService,
	cfg *config.Config,
	rdb *redis.Client,
	logger *slog.Logger,
) *MedicalRecordHandler {
	return &MedicalRecordHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

func (h *MedicalRecordHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))

		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})

	return r
}

func (h *MedicalRecordHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	isAuth := false
	for _, role := range roles {
		if role == "doctor" || role == "admin" {
			isAuth = true
			break
		}
	}
	if !isAuth {
		httpcallForbidden(w, "unauthorized access to create medical record")
		return
	}

	var req dto.CreateMedicalRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "malformed JSON request body")
		return
	}

	if err := validator.ValidateCreate(req); err != nil {
		httpcallBadRequest(w, err.Error())
		return
	}

	ip, ua := getClientContext(r)
	resp, err := h.svc.Create(r.Context(), userID, roles, ip, ua, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnauthorized):
			httpcallForbidden(w, "unauthorized access to create medical record")
		case errors.Is(err, service.ErrTreatmentRelationshipRequired):
			httpcallForbidden(w, err.Error())
		default:
			h.logger.Error("failed to create medical record", "error", err)
			httpresponse.InternalError(w)
		}
		return
	}

	httpresponse.JSON(w, http.StatusCreated, httpresponse.Response{
		Success: true,
		Data:    resp,
	})
}

func (h *MedicalRecordHandler) GetByID(w http.ResponseWriter, r *http.Request) {
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
		httpcallBadRequest(w, "invalid medical record UUID format")
		return
	}

	ip, ua := getClientContext(r)
	resp, err := h.svc.GetByID(r.Context(), userID, roles, ip, ua, id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			httpresponse.Error(w, http.StatusNotFound, "RECORD_NOT_FOUND", "medical record not found")
		case errors.Is(err, service.ErrUnauthorized):
			httpcallForbidden(w, "unauthorized access to medical record")
		case errors.Is(err, service.ErrTreatmentRelationshipRequired):
			httpcallForbidden(w, err.Error())
		default:
			h.logger.Error("failed to get medical record", "error", err)
			httpcallInternalError(w)
		}
		return
	}

	httpresponse.Success(w, resp)
}

func (h *MedicalRecordHandler) List(w http.ResponseWriter, r *http.Request) {
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

	patientIDFilter := r.URL.Query().Get("patient_id")
	recordTypeFilter := r.URL.Query().Get("record_type")

	var patientIDPtr *string
	if patientIDFilter != "" {
		patientIDPtr = &patientIDFilter
	}
	var recordTypePtr *string
	if recordTypeFilter != "" {
		recordTypePtr = &recordTypeFilter
	}

	filter := dto.ListMedicalRecordsFilter{
		PatientID:  patientIDPtr,
		RecordType: recordTypePtr,
	}

	ip, ua := getClientContext(r)
	resp, err := h.svc.List(r.Context(), userID, roles, ip, ua, filter)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPatientIDRequired):
			httpcallBadRequest(w, err.Error())
		case errors.Is(err, service.ErrTreatmentRelationshipRequired):
			httpcallForbidden(w, err.Error())
		case errors.Is(err, service.ErrUnauthorized):
			httpcallForbidden(w, "unauthorized access to medical records")
		default:
			h.logger.Error("failed to list medical records", "error", err)
			httpcallInternalError(w)
		}
		return
	}

	httpcallSuccess(w, resp)
}

func (h *MedicalRecordHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	isAuth := false
	for _, role := range roles {
		if role == "doctor" || role == "admin" {
			isAuth = true
			break
		}
	}
	if !isAuth {
		httpcallForbidden(w, "unauthorized access to update medical record")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpcallBadRequest(w, "invalid medical record UUID format")
		return
	}

	var req dto.UpdateMedicalRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "malformed JSON request body")
		return
	}

	if err := validator.ValidateUpdate(req); err != nil {
		httpcallBadRequest(w, err.Error())
		return
	}

	ip, ua := getClientContext(r)
	resp, err := h.svc.Update(r.Context(), userID, roles, ip, ua, id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			httpresponse.Error(w, http.StatusNotFound, "RECORD_NOT_FOUND", "medical record not found")
		case errors.Is(err, service.ErrOnlyCreatorCanModify):
			httpcallForbidden(w, err.Error())
		case errors.Is(err, service.ErrUnauthorized):
			httpcallForbidden(w, "unauthorized access to update medical record")
		default:
			h.logger.Error("failed to update medical record", "error", err)
			httpcallInternalError(w)
		}
		return
	}

	httpresponse.Success(w, resp)
}

func (h *MedicalRecordHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	isAdmin := false
	for _, role := range roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		httpcallForbidden(w, "unauthorized access to delete medical record")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpcallBadRequest(w, "invalid medical record UUID format")
		return
	}

	ip, ua := getClientContext(r)
	err = h.svc.Delete(r.Context(), userID, roles, ip, ua, id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			httpcallNotFound(w, "RECORD_NOT_FOUND", "medical record not found")
		case errors.Is(err, service.ErrUnauthorized):
			httpcallForbidden(w, "unauthorized access to delete medical record")
		default:
			h.logger.Error("failed to delete medical record", "error", err)
			httpcallInternalError(w)
		}
		return
	}

	httpresponse.SuccessWithMessage(w, "Medical record soft-deleted successfully", nil)
}

func getClientContext(r *http.Request) (string, string) {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	ua := r.UserAgent()
	return ip, ua
}

func httpcallUnauthorized(w http.ResponseWriter) {
	httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

func httpcallForbidden(w http.ResponseWriter, message string) {
	httpcallError(w, http.StatusForbidden, "FORBIDDEN", message)
}

func httpcallBadRequest(w http.ResponseWriter, message string) {
	httpcallError(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

func httpcallNotFound(w http.ResponseWriter, code, message string) {
	httpcallError(w, http.StatusNotFound, code, message)
}

func httpcallInternalError(w http.ResponseWriter) {
	httpresponse.InternalError(w)
}

func httpcallSuccess(w http.ResponseWriter, data any) {
	httpresponse.Success(w, data)
}

func httpcallError(w http.ResponseWriter, status int, code, message string) {
	httpresponse.Error(w, status, code, message)
}
