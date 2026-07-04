package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/timurdianradhasejati/telemed_hub/internal/auth/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
)

type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) CreateUserWithRoleAndProfile(ctx context.Context, user *model.User, roleName string) error {
	args := m.Called(ctx, user, roleName)
	return args.Error(0)
}

func (m *MockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAuthRepository) SaveRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	args := m.Called(ctx, token)
	token.ID = uuid.New()
	return args.Error(0)
}

func (m *MockAuthRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RefreshToken), args.Error(1)
}

func (m *MockAuthRepository) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockAuthRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAuthRepository) GetActiveRefreshTokens(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.RefreshToken), args.Error(1)
}

func TestAuthService_Register(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	logger := slogNew()
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:    "Y2hhbmdlLW1lLXRvLWEtMzItYnl0ZS1iYXNlNjQtZW5jb2RlZC1zZWNyZXQ=", // Valid base64
			AccessTTL: 15 * time.Minute,
		},
	}

	svc := NewAuthService(mockRepo, nil, cfg, logger)

	t.Run("Success Registration", func(t *testing.T) {
		req := dto.RegisterRequest{
			FullName: "Test User",
			Email:    "test@user.com",
			Password: "Password123!",
			Role:     "patient",
		}

		mockRepo.On("CreateUserWithRoleAndProfile", mock.Anything, mock.Anything, "patient").Return(nil).Once()

		resp, err := svc.Register(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "test@user.com", resp.Email)
		assert.Equal(t, "patient", resp.Role)
		assert.NotEmpty(t, resp.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Validation Failure", func(t *testing.T) {
		req := dto.RegisterRequest{
			FullName: "",
			Email:    "invalid-email",
			Password: "123",
			Role:     "invalid-role",
		}

		resp, err := svc.Register(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestAuthService_Login(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	logger := slogNew()
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:    "Y2hhbmdlLW1lLXRvLWEtMzItYnl0ZS1iYXNlNjQtZW5jb2RlZC1zZWNyZXQ=",
			AccessTTL: 15 * time.Minute,
		},
	}

	svc := NewAuthService(mockRepo, nil, cfg, logger)

	password := "SecurePass123!"
	hashedPassword, err := model.HashPassword(password)
	require.NoError(t, err)

	user := &model.User{
		ID:           uuid.New(),
		Email:        "login@test.com",
		PasswordHash: hashedPassword,
		FullName:     "Login User",
		Status:       "active",
	}

	t.Run("Success Login", func(t *testing.T) {
		req := dto.LoginRequest{
			Email:    "login@test.com",
			Password: password,
		}

		mockRepo.On("GetUserByEmail", mock.Anything, "login@test.com").Return(user, nil).Once()
		mockRepo.On("GetUserRoles", mock.Anything, user.ID).Return([]string{"patient"}, nil).Once()
		mockRepo.On("SaveRefreshToken", mock.Anything, mock.Anything).Return(nil).Once()

		resp, err := svc.Login(context.Background(), req, "127.0.0.1", "Mozilla")
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed Login - Invalid Password", func(t *testing.T) {
		req := dto.LoginRequest{
			Email:    "login@test.com",
			Password: "wrong_password",
		}

		mockRepo.On("GetUserByEmail", mock.Anything, "login@test.com").Return(user, nil).Once()

		resp, err := svc.Login(context.Background(), req, "127.0.0.1", "Mozilla")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Nil(t, resp)
		mockRepo.AssertExpectations(t)
	})
}

// slogNew is a helper to get a dummy logger for tests
func slogNew() *slog.Logger {
	return logger.Setup("error")
}
