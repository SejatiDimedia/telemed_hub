package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/validator"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type WalletHandler struct {
	svc    service.WalletService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewWalletHandler(
	svc service.WalletService,
	cfg *config.Config,
	rdb *redis.Client,
	logger *slog.Logger,
) *WalletHandler {
	return &WalletHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

func (h *WalletHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))

		r.Get("/", h.GetBalance)
		r.Post("/top-up", h.TopUp)
		r.Get("/transactions", h.ListTransactions)
	})

	return r
}

func (h *WalletHandler) checkPatientAuth(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return uuid.Nil, false
	}

	roles, err := middleware.GetUserRoles(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return uuid.Nil, false
	}

	authorized := false
	for _, role := range roles {
		if role == "patient" {
			authorized = true
			break
		}
	}

	if !authorized {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "unauthorized access to patient wallet")
		return uuid.Nil, false
	}

	return userID, true
}

func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkPatientAuth(w, r)
	if !ok {
		return
	}

	resp, err := h.svc.GetBalanceDetails(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get wallet balance details", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

func (h *WalletHandler) TopUp(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkPatientAuth(w, r)
	if !ok {
		return
	}

	var req dto.TopUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "malformed JSON request body")
		return
	}

	if err := validator.ValidateTopUp(req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	var keyPtr *string
	if idempotencyKey != "" {
		keyPtr = &idempotencyKey
	}

	resp, err := h.svc.TopUp(r.Context(), userID, req.Amount, keyPtr)
	if err != nil {
		if errors.Is(err, service.ErrMaxTopUpExceeded) {
			httpresponse.Error(w, http.StatusBadRequest, "MAX_TOPUP_EXCEEDED", err.Error())
			return
		}
		h.logger.Error("failed to process wallet top-up", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.JSON(w, http.StatusCreated, httpresponse.Response{
		Success: true,
		Data:    resp,
	})
}

func (h *WalletHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkPatientAuth(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()

	var typeFilter *string
	if t := q.Get("type"); t != "" {
		typeFilter = &t
	}

	page := 1
	if pStr := q.Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if lStr := q.Get("page_size"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	txs, totalItems, err := h.svc.ListTransactions(r.Context(), userID, typeFilter, page, limit)
	if err != nil {
		h.logger.Error("failed to list wallet transactions", "error", err)
		httpcallInternalError(w)
		return
	}

	totalPages := (totalItems + limit - 1) / limit

	httpresponse.JSON(w, http.StatusOK, httpresponse.Response{
		Success: true,
		Data:    txs,
		Pagination: &httpresponse.PaginationInfo{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	})
}

func httpcallInternalError(w http.ResponseWriter) {
	httpresponse.InternalError(w)
}
