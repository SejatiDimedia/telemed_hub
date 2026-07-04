package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/validator"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RegisterResponse), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	args := m.Called(ctx, req, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AuthResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, req dto.RefreshRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	args := m.Called(ctx, req, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AuthResponse), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, userID uuid.UUID, rawRefreshToken string, allDevices bool) error {
	args := m.Called(ctx, userID, rawRefreshToken, allDevices)
	return args.Error(0)
}

func (m *MockAuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*dto.UserResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UserResponse), args.Error(1)
}

func TestAuthHandler_Register(t *testing.T) {
	mockSvc := new(MockAuthService)
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewAuthHandler(mockSvc, cfg, nil, log)

	r := chi.NewRouter()
	r.Post("/auth/register", h.Register)

	t.Run("Success Register", func(t *testing.T) {
		reqBody := dto.RegisterRequest{
			FullName: "Rina Wijaya",
			Email:    "rina@test.com",
			Password: "Password123!",
			Role:     "patient",
		}
		expectedResp := &dto.RegisterResponse{
			ID:    uuid.New().String(),
			Email: "rina@test.com",
			Role:  "patient",
		}

		mockSvc.On("Register", mock.Anything, reqBody).Return(expectedResp, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var envelope struct {
			Success bool                  `json:"success"`
			Data    dto.RegisterResponse `json:"data"`
		}
		err := json.NewDecoder(rec.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Equal(t, expectedResp.Email, envelope.Data.Email)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Conflict Email", func(t *testing.T) {
		reqBody := dto.RegisterRequest{
			FullName: "Rina Wijaya",
			Email:    "rina@test.com",
			Password: "Password123!",
			Role:     "patient",
		}

		mockSvc.On("Register", mock.Anything, reqBody).Return(nil, repository.ErrEmailConflict).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		var envelope struct {
			Success   bool   `json:"success"`
			ErrorCode string `json:"error_code"`
		}
		err := json.NewDecoder(rec.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.False(t, envelope.Success)
		assert.Equal(t, "CONFLICT", envelope.ErrorCode)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Validation Error", func(t *testing.T) {
		reqBody := dto.RegisterRequest{
			FullName: "Rina",
			Email:    "invalid-email",
			Password: "123",
			Role:     "patient",
		}

		valErr := validator.ValidationErrors{
			{Field: "email", Issue: "must be a valid email address"},
			{Field: "password", Issue: "must be at least 8 characters long"},
		}

		mockSvc.On("Register", mock.Anything, reqBody).Return(nil, valErr).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var envelope struct {
			Success   bool   `json:"success"`
			ErrorCode string `json:"error_code"`
		}
		err := json.NewDecoder(rec.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.False(t, envelope.Success)
		assert.Equal(t, "VALIDATION_ERROR", envelope.ErrorCode)
		mockSvc.AssertExpectations(t)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	mockSvc := new(MockAuthService)
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewAuthHandler(mockSvc, cfg, nil, log)

	r := chi.NewRouter()
	r.Post("/auth/login", h.Login)

	t.Run("Success Login", func(t *testing.T) {
		reqBody := dto.LoginRequest{
			Email:    "rina@test.com",
			Password: "Password123!",
		}
		expectedResp := &dto.AuthResponse{
			AccessToken:  "access_token_jwt",
			RefreshToken: "refresh_token_string",
			ExpiresIn:    900,
			TokenType:    "Bearer",
		}

		mockSvc.On("Login", mock.Anything, reqBody, mock.Anything, mock.Anything).Return(expectedResp, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var envelope struct {
			Success bool             `json:"success"`
			Data    dto.AuthResponse `json:"data"`
		}
		err := json.NewDecoder(rec.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Equal(t, expectedResp.AccessToken, envelope.Data.AccessToken)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Unauthorized Credentials", func(t *testing.T) {
		reqBody := dto.LoginRequest{
			Email:    "rina@test.com",
			Password: "wrong_password",
		}

		mockSvc.On("Login", mock.Anything, reqBody, mock.Anything, mock.Anything).Return(nil, service.ErrInvalidCredentials).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var envelope struct {
			Success   bool   `json:"success"`
			ErrorCode string `json:"error_code"`
			Error     string `json:"error"`
		}
		err := json.NewDecoder(rec.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.False(t, envelope.Success)
		assert.Equal(t, "UNAUTHORIZED", envelope.ErrorCode)
		assert.Equal(t, "Invalid email or password", envelope.Error) // Generic message
		mockSvc.AssertExpectations(t)
	})
}
