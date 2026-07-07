package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type AdminHandler struct {
	svc    service.AdminService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewAdminHandler(
	svc service.AdminService,
	cfg *config.Config,
	rdb *redis.Client,
	logger *slog.Logger,
) *AdminHandler {
	return &AdminHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

func (h *AdminHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))

		r.Get("/audit-logs", h.ListAuditLogs)
	})

	return r
}

func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
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

	actorID := r.URL.Query().Get("actor_id")
	action := r.URL.Query().Get("action")
	targetType := r.URL.Query().Get("target_type")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var actorIDPtr *string
	if actorID != "" {
		actorIDPtr = &actorID
	}
	var actionPtr *string
	if action != "" {
		actionPtr = &action
	}
	var targetTypePtr *string
	if targetType != "" {
		targetTypePtr = &targetType
	}
	var fromPtr *string
	if from != "" {
		fromPtr = &from
	}
	var toPtr *string
	if to != "" {
		toPtr = &to
	}

	filter := dto.ListAuditLogsFilter{
		ActorID:    actorIDPtr,
		Action:     actionPtr,
		TargetType: targetTypePtr,
		From:       fromPtr,
		To:         toPtr,
		Page:       page,
		Limit:      limit,
	}

	resp, total, err := h.svc.ListAuditLogs(r.Context(), userID, roles, filter)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized access to admin audit logs")
			return
		}
		h.logger.Error("failed to list audit logs", "error", err)
		httpcallInternalError(w)
		return
	}

	totalPages := (total + limit - 1) / limit
	httpcallSuccessWithPagination(w, resp, page, limit, total, totalPages)
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

func httpcallSuccessWithPagination(w http.ResponseWriter, data any, page, limit, totalItems, totalPages int) {
	httpresponse.JSON(w, http.StatusOK, httpresponse.Response{
		Success: true,
		Data:    data,
		Pagination: &httpresponse.PaginationInfo{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	})
}
