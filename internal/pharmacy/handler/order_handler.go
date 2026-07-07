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
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/validator"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type OrderHandler struct {
	svc    service.OrderService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewOrderHandler(
	svc service.OrderService,
	cfg *config.Config,
	rdb *redis.Client,
	logger *slog.Logger,
) *OrderHandler {
	return &OrderHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

func (h *OrderHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))

		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}/status", h.UpdateStatus)
		r.Post("/{id}/cancel", h.Cancel)
	})

	return r
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	// Only patients can place orders
	isPatient := false
	for _, role := range roles {
		if role == "patient" {
			isPatient = true
			break
		}
	}
	if !isPatient {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "only patients can place orders")
		return
	}

	var req dto.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "malformed JSON request body")
		return
	}

	if err := validator.ValidateCreateOrder(req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	var keyPtr *string
	if idempotencyKey != "" {
		keyPtr = &idempotencyKey
	}

	resp, err := h.svc.Create(r.Context(), userID, req, keyPtr)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInsufficientBalance):
			httpresponse.Error(w, http.StatusUnprocessableEntity, "INSUFFICIENT_BALANCE", "insufficient wallet balance for this order")
		case errors.Is(err, service.ErrOutOfStock):
			httpcallUnprocessableEntity(w, "OUT_OF_STOCK", "one or more medicines are out of stock")
		case errors.Is(err, service.ErrUnauthorized):
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized access to prescription")
		default:
			h.logger.Error("failed to create order", "error", err)
			httpresponse.InternalError(w)
		}
		return
	}

	httpresponse.JSON(w, http.StatusCreated, httpresponse.Response{
		Success: true,
		Data:    resp,
	})
}

func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
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
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_ORDER_ID", "invalid order UUID format")
		return
	}

	resp, err := h.svc.GetByID(r.Context(), id, userID, roles)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			httpresponse.Error(w, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found")
		case errors.Is(err, service.ErrUnauthorized):
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized access to order details")
		default:
			h.logger.Error("failed to get order", "error", err)
			httpresponse.InternalError(w)
		}
		return
	}

	httpresponse.Success(w, resp)
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
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
		h.logger.Error("failed to list orders", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	// Only pharmacy staff or admin can update order status
	isStaff := false
	for _, role := range roles {
		if role == "pharmacy_staff" || role == "admin" {
			isStaff = true
			break
		}
	}
	if !isStaff {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized: only pharmacy staff can transition order status")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_ORDER_ID", "invalid order UUID format")
		return
	}

	var req dto.UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "malformed JSON request body")
		return
	}

	if err := validator.ValidateUpdateStatus(req); err != nil {
		httpcallBadRequest(w, err.Error())
		return
	}

	resp, err := h.svc.UpdateStatus(r.Context(), userID, id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			httpresponse.Error(w, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found")
		case errors.Is(err, service.ErrInvalidStatusTransition):
			httpcallUnprocessableEntity(w, "INVALID_STATUS_TRANSITION", "invalid order status transition lifecycle state change")
		default:
			h.logger.Error("failed to update order status", "error", err)
			httpresponse.InternalError(w)
		}
		return
	}

	httpresponse.Success(w, resp)
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpcallUnauthorized(w)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpcallBadRequest(w, "invalid order UUID format")
		return
	}

	err = h.svc.Cancel(r.Context(), userID, id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			httpresponse.Error(w, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found")
		case errors.Is(err, service.ErrUnauthorized):
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized: not the owner of this order")
		case errors.Is(err, service.ErrOrderCannotBeCancelled):
			httpcallUnprocessableEntity(w, "ORDER_NOT_CANCELLABLE", err.Error())
		default:
			h.logger.Error("failed to cancel order", "error", err)
			httpresponse.InternalError(w)
		}
		return
	}

	httpcallSuccessWithMessage(w, "Order cancelled successfully", nil)
}

func httpcallUnauthorized(w http.ResponseWriter) {
	httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

func httpcallBadRequest(w http.ResponseWriter, msg string) {
	httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", msg)
}

func httpcallUnprocessableEntity(w http.ResponseWriter, code, msg string) {
	httpresponse.Error(w, http.StatusUnprocessableEntity, code, msg)
}

func httpcallSuccessWithMessage(w http.ResponseWriter, message string, data any) {
	httpresponse.SuccessWithMessage(w, message, data)
}
