package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/validator"
	"github.com/timurdianradhasejati/telemed_hub/pkg/httpresponse"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type AuthHandler struct {
	svc    service.AuthService
	cfg    *config.Config
	rdb    *redis.Client
	logger *slog.Logger
}

func NewAuthHandler(svc service.AuthService, cfg *config.Config, rdb *redis.Client, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		svc:    svc,
		cfg:    cfg,
		rdb:    rdb,
		logger: logger,
	}
}

// Routes wires auth endpoints.
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public routes
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(h.cfg, h.rdb))
		r.Post("/logout", h.Logout)
		r.Get("/me", h.Me)
	})

	return r
}

// Register handler.
// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	resp, err := h.svc.Register(r.Context(), req)
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
		if errors.Is(err, repository.ErrEmailConflict) {
			httpresponse.Error(w, http.StatusConflict, "CONFLICT", "Email is already registered")
			return
		}
		h.logger.Error("registration handler error", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Created(w, resp)
}

// Login handler.
// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Rate Limiting (checked via middleware ideally, but let's enforce it here as well for self-contained robustness)
	if h.rdb != nil {
		clientIP := getClientIP(r)
		rateLimitKey := "rate:ip:" + clientIP + ":auth_login"
		count, err := h.rdb.Incr(r.Context(), rateLimitKey).Result()
		if err == nil {
			if count == 1 {
				_ = h.rdb.Expire(r.Context(), rateLimitKey, 60*time.Second)
			}
			if count > 5 { // 5 requests per minute limit
				w.Header().Set("Retry-After", "60")
				httpresponse.Error(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many login attempts. Please try again later.")
				return
			}
		}
	}

	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	ipAddress := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	resp, err := h.svc.Login(r.Context(), req, ipAddress, userAgent)
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
		if errors.Is(err, service.ErrInvalidCredentials) {
			// Timing-safe generic credentials error
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password")
			return
		}
		if errors.Is(err, service.ErrSuspendedUser) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Your account has been suspended")
			return
		}
		h.logger.Error("login handler error", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

// Refresh handler.
// POST /auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	ipAddress := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	resp, err := h.svc.RefreshToken(r.Context(), req, ipAddress, userAgent)
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
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrExpiredToken) {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired refresh token")
			return
		}
		if errors.Is(err, service.ErrSuspendedUser) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Your account has been suspended")
			return
		}
		h.logger.Error("refresh handler error", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handler.
// POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	allDevicesHeader := r.Header.Get("X-All-Devices")
	allDevices := strings.ToLower(allDevicesHeader) == "true"

	var rawToken string
	if !allDevices {
		var req LogoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
			httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Refresh token is required in body for logout")
			return
		}
		rawToken = req.RefreshToken
	}

	err = h.svc.Logout(r.Context(), userID, rawToken, allDevices)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid session token")
			return
		}
		h.logger.Error("logout handler error", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.SuccessWithMessage(w, "logged out successfully", nil)
}

// Me handler.
// GET /auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthenticated")
		return
	}

	resp, err := h.svc.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		h.logger.Error("me handler error", "error", err)
		httpresponse.InternalError(w)
		return
	}

	httpresponse.Success(w, resp)
}

// --- helpers ---

func getClientIP(r *http.Request) string {
	// Simple IP extraction
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return r.RemoteAddr
}
