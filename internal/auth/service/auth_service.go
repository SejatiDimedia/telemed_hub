package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/mapper"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/validator"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSuspendedUser      = errors.New("user account is suspended")
	ErrInvalidToken       = errors.New("invalid or revoked refresh token")
	ErrExpiredToken       = errors.New("refresh token has expired")
)

type AuthServiceImpl struct {
	repo   repository.AuthRepository
	rdb    *redis.Client
	cfg    *config.Config
	logger *slog.Logger
}

func NewAuthService(repo repository.AuthRepository, rdb *redis.Client, cfg *config.Config, logger *slog.Logger) *AuthServiceImpl {
	return &AuthServiceImpl{
		repo:   repo,
		rdb:    rdb,
		cfg:    cfg,
		logger: logger,
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	if err := validator.ValidateRegister(req); err != nil {
		return nil, err
	}

	// Hash password using Argon2id
	passHash, err := model.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		ID:           uuid.New(),
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: passHash,
		FullName:     strings.TrimSpace(req.FullName),
		IsVerified:   false,
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	err = s.repo.CreateUserWithRoleAndProfile(ctx, user, req.Role)
	if err != nil {
		if errors.Is(err, repository.ErrEmailConflict) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("user registered successfully", "user_id", user.ID, "role", req.Role)

	resp := mapper.ToRegisterResponse(user, req.Role)
	return &resp, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	if err := validator.ValidateLogin(req); err != nil {
		return nil, err
	}

	// 1. Fetch user by email
	user, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))

	// Timing attack prevention: compare dummy hash if user not found
	dummyHash := "$argon2id$v=19$m=65536,t=1,p=4$c29tZXNhbHQ$dGVzdF9kdW1teV9oYXNoX2Zvcl90aW1pbmc="
	if err != nil {
		_, _ = model.ComparePassword(req.Password, dummyHash)
		return nil, ErrInvalidCredentials
	}

	// 2. Verify password hash
	match, err := model.ComparePassword(req.Password, user.PasswordHash)
	if err != nil || !match {
		return nil, ErrInvalidCredentials
	}

	// 3. Enforce suspension check
	if user.Status == "suspended" {
		s.logger.Warn("login attempt from suspended user", "user_id", user.ID)
		return nil, ErrSuspendedUser
	}

	// 4. Generate Tokens
	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	accessToken, err := s.generateAccessToken(user, roles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	rawRefreshToken, dbRefreshToken, err := s.createRefreshTokenRecord(user.ID, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	err = s.repo.SaveRefreshToken(ctx, dbRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	s.logger.Info("user logged in successfully", "user_id", user.ID)

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int(s.cfg.JWT.AccessTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthServiceImpl) RefreshToken(ctx context.Context, req dto.RefreshRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	if err := validator.ValidateRefresh(req); err != nil {
		return nil, err
	}

	// Parse composite token: userIDStr.randomToken
	parts := strings.SplitN(req.RefreshToken, ".", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}

	rawTokenPart := parts[1]

	// 1. Fetch user to verify status
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.Status == "suspended" {
		return nil, ErrSuspendedUser
	}

	// 2. Fetch active refresh tokens for the user
	activeTokens, err := s.repo.GetActiveRefreshTokens(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active refresh tokens: %w", err)
	}

	var matchedToken *model.RefreshToken
	for _, t := range activeTokens {
		match, err := model.ComparePassword(rawTokenPart, t.TokenHash)
		if err == nil && match {
			matchedToken = t
			break
		}
	}

	if matchedToken == nil {
		s.logger.Warn("refresh token mismatch or reuse detected", "user_id", userID)
		// Suspicious concurrent refresh reuse detection: revoke all tokens if client presents an invalid/revoked token
		_ = s.repo.RevokeAllUserRefreshTokens(ctx, userID)
		return nil, ErrInvalidToken
	}

	// Check expiry (double guard, database query already filtered expired tokens, but safety first)
	if time.Now().UTC().After(matchedToken.ExpiresAt) {
		return nil, ErrExpiredToken
	}

	// 3. Rotate tokens: Revoke old token and issue new ones
	err = s.repo.RevokeRefreshToken(ctx, matchedToken.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke old token: %w", err)
	}

	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	newAccessToken, err := s.generateAccessToken(user, roles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRawRefreshToken, newDbRefreshToken, err := s.createRefreshTokenRecord(user.ID, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	err = s.repo.SaveRefreshToken(ctx, newDbRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to save new refresh token: %w", err)
	}

	s.logger.Info("token refreshed successfully", "user_id", user.ID)

	return &dto.AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRawRefreshToken,
		ExpiresIn:    int(s.cfg.JWT.AccessTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthServiceImpl) Logout(ctx context.Context, userID uuid.UUID, rawRefreshToken string, allDevices bool) error {
	if allDevices {
		// Revoke all tokens in DB
		err := s.repo.RevokeAllUserRefreshTokens(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to revoke all refresh tokens: %w", err)
		}

		// Add user to Redis blocklist for forced access token revocation
		// Key TTL = access token TTL (15 minutes)
		blocklistKey := fmt.Sprintf("blocklist:user:%s", userID.String())
		err = s.rdb.Set(ctx, blocklistKey, "revoked", s.cfg.JWT.AccessTTL).Err()
		if err != nil {
			s.logger.Error("failed to set blocklist key in redis", "error", err, "user_id", userID)
		}

		s.logger.Info("user logged out from all devices", "user_id", userID)
		return nil
	}

	// Single device logout: split token to verify hash and revoke
	parts := strings.SplitN(rawRefreshToken, ".", 2)
	if len(parts) != 2 {
		return ErrInvalidToken
	}

	rawTokenPart := parts[1]
	activeTokens, err := s.repo.GetActiveRefreshTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get active tokens for logout: %w", err)
	}

	for _, t := range activeTokens {
		match, err := model.ComparePassword(rawTokenPart, t.TokenHash)
		if err == nil && match {
			err = s.repo.RevokeRefreshToken(ctx, t.ID)
			if err != nil {
				return fmt.Errorf("failed to revoke refresh token: %w", err)
			}
			s.logger.Info("user logged out successfully", "user_id", userID)
			return nil
		}
	}

	return ErrInvalidToken
}

func (s *AuthServiceImpl) GetUserByID(ctx context.Context, id uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	roles, err := s.repo.GetUserRoles(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	resp := mapper.ToUserResponse(user, roles)
	return &resp, nil
}

// --- helpers ---

func (s *AuthServiceImpl) getSigningKey() ([]byte, error) {
	if s.cfg.JWT.Secret == "" {
		return nil, errors.New("JWT secret is not configured")
	}
	key, err := base64.StdEncoding.DecodeString(s.cfg.JWT.Secret)
	if err != nil {
		// Fallback to raw secret bytes for dev/test keys
		return []byte(s.cfg.JWT.Secret), nil
	}
	return key, nil
}

func (s *AuthServiceImpl) generateAccessToken(user *model.User, roles []string) (string, error) {
	signingKey, err := s.getSigningKey()
	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"roles": roles,
		"iat":   time.Now().UTC().Unix(),
		"exp":   time.Now().UTC().Add(s.cfg.JWT.AccessTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(signingKey)
}

func (s *AuthServiceImpl) createRefreshTokenRecord(userID uuid.UUID, ipAddress, userAgent string) (string, *model.RefreshToken, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random bytes for refresh token: %w", err)
	}

	rawRandomToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	compositeToken := fmt.Sprintf("%s.%s", userID.String(), rawRandomToken)

	// Hash the raw token part for DB storage (using faster parameters for token hashing, but DefaultParams is fine)
	hashedToken, err := model.HashPassword(rawRandomToken)
	if err != nil {
		return "", nil, fmt.Errorf("failed to hash refresh token: %w", err)
	}

	expiresAt := time.Now().AddDate(0, 0, s.cfg.JWT.RefreshTTLDays).UTC()

	var ip, ua *string
	if ipAddress != "" {
		ip = &ipAddress
	}
	if userAgent != "" {
		ua = &userAgent
	}

	dbToken := &model.RefreshToken{
		UserID:    userID,
		TokenHash: hashedToken,
		ExpiresAt: expiresAt,
		UserAgent: ua,
		IPAddress: ip,
		CreatedAt: time.Now().UTC(),
	}

	return compositeToken, dbToken, nil
}
