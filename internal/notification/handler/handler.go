package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type NotificationHandler struct {
	svc service.NotificationService
	cfg *config.Config
	rdb *redis.Client
}

func NewNotificationHandler(svc service.NotificationService, cfg *config.Config, rdb *redis.Client) *NotificationHandler {
	return &NotificationHandler{
		svc: svc,
		cfg: cfg,
		rdb: rdb,
	}
}

func (h *NotificationHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))
	r.Get("/", h.List)
	r.Post("/{id}/read", h.MarkAsRead)
	return r
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	var statusPtr *string
	if statusFilter == "unread" {
		sentStr := "sent"
		statusPtr = &sentStr
	} else if statusFilter == "read" {
		readStr := "read"
		statusPtr = &readStr
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	resp, total, err := h.svc.ListNotifications(r.Context(), userID, statusPtr, page, limit)
	if err != nil {
		httpcallInternalError(w)
		return
	}

	totalPages := (total + limit - 1) / limit
	httpresponse.JSON(w, http.StatusOK, httpresponse.Response{
		Success: true,
		Data:    resp,
		Pagination: &httpcallPaginationInfo{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	})
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpcallBadRequest(w, "invalid notification UUID format")
		return
	}

	err = h.svc.MarkAsRead(r.Context(), userID, id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			httpcallNotFound(w, "NOTIFICATION_NOT_FOUND", "notification not found")
		case errors.Is(err, service.ErrUnauthorized):
			httpcallForbidden(w, "unauthorized to read this notification")
		default:
			httpcallInternalError(w)
		}
		return
	}

	httpcallSuccess(w, "Notification marked as read successfully")
}

func httpcallUnauthorized(w http.ResponseWriter) {
	httpcallError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

func httpcallForbidden(w http.ResponseWriter, msg string) {
	httpcallError(w, http.StatusForbidden, "FORBIDDEN", msg)
}

func httpcallBadRequest(w http.ResponseWriter, msg string) {
	httpcallError(w, http.StatusBadRequest, "BAD_REQUEST", msg)
}

func httpcallNotFound(w http.ResponseWriter, code, msg string) {
	httpcallError(w, http.StatusNotFound, code, msg)
}

func httpcallInternalError(w http.ResponseWriter) {
	httpcallError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "an unexpected error occurred")
}

func httpcallError(w http.ResponseWriter, status int, code, message string) {
	httpcallResponse := httpresponse.Response{
		Success:   false,
		Error:     message,
		ErrorCode: code,
	}
	httpresponse.JSON(w, status, httpcallResponse)
}

func httpcallSuccess(w http.ResponseWriter, message string) {
	httpresponse.JSON(w, http.StatusOK, httpresponse.Response{
		Success: true,
		Data:    map[string]any{"message": message},
	})
}

type httpcallPaginationInfo = httpresponse.PaginationInfo
