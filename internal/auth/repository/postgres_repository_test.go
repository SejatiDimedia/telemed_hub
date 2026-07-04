package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/timurdianradhasejati/telemed_hub/internal/auth/model"
)

func setupTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	// Spin up PostgreSQL test container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
	)
	require.NoError(t, err)

	connURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create pgx connection pool with retry loop to handle transient startup state
	var pool *pgxpool.Pool
	var pingErr error
	for i := 0; i < 20; i++ {
		pool, err = pgxpool.New(ctx, connURL)
		if err == nil {
			pingErr = pool.Ping(ctx)
			if pingErr == nil {
				break
			}
			pool.Close()
		} else {
			pingErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.NoError(t, pingErr, "failed to ping database after retries")

	// Run migrations programmatically
	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	require.NoError(t, err)

	m, err := migrate.New("file://"+migrationsPath, connURL)
	require.NoError(t, err)

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}

	cleanup := func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return pool, cleanup
}

func TestPostgresRepository_UserOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	// Test 1: Create Patient User
	user1 := &model.User{
		ID:           uuid.New(),
		Email:        "patient@test.com",
		PasswordHash: "some_argon_hash",
		FullName:     "John Patient",
		IsVerified:   true,
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	err := repo.CreateUserWithRoleAndProfile(ctx, user1, "patient")
	require.NoError(t, err)

	// Verify User retrieval by Email
	retrievedUser, err := repo.GetUserByEmail(ctx, "patient@test.com")
	require.NoError(t, err)
	assert.Equal(t, user1.ID, retrievedUser.ID)
	assert.Equal(t, user1.FullName, retrievedUser.FullName)

	// Verify User roles
	roles, err := repo.GetUserRoles(ctx, user1.ID)
	require.NoError(t, err)
	assert.Contains(t, roles, "patient")

	// Test 2: Unique Constraint Conflict
	duplicateUser := &model.User{
		ID:           uuid.New(),
		Email:        "patient@test.com", // Duplicate
		PasswordHash: "hash",
		FullName:     "Duplicate Patient",
		IsVerified:   true,
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	err = repo.CreateUserWithRoleAndProfile(ctx, duplicateUser, "patient")
	assert.ErrorIs(t, err, ErrEmailConflict)

	// Test 3: Get non-existent user
	nonExistentUser, err := repo.GetUserByEmail(ctx, "doesnotexist@test.com")
	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, nonExistentUser)
}

func TestPostgresRepository_TokenOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	// Create user first
	user := &model.User{
		ID:           uuid.New(),
		Email:        "user@test.com",
		PasswordHash: "hash",
		FullName:     "User",
		IsVerified:   true,
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	err := repo.CreateUserWithRoleAndProfile(ctx, user, "patient")
	require.NoError(t, err)

	// Save Refresh Token
	userAgent := "Mozilla/5.0"
	ipAddr := "127.0.0.1"
	token := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: "hashed_refresh_token_123",
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
		UserAgent: &userAgent,
		IPAddress: &ipAddr,
		CreatedAt: time.Now().UTC(),
	}
	err = repo.SaveRefreshToken(ctx, token)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, token.ID)

	// Get Token by Hash
	retrievedToken, err := repo.GetRefreshTokenByHash(ctx, "hashed_refresh_token_123")
	require.NoError(t, err)
	assert.Equal(t, token.ID, retrievedToken.ID)
	assert.Nil(t, retrievedToken.RevokedAt)

	// Get Active Tokens
	activeTokens, err := repo.GetActiveRefreshTokens(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, activeTokens, 1)

	// Revoke Single Token
	err = repo.RevokeRefreshToken(ctx, token.ID)
	require.NoError(t, err)

	// Verify Single Token Revoked
	retrievedToken, err = repo.GetRefreshTokenByHash(ctx, "hashed_refresh_token_123")
	require.NoError(t, err)
	assert.NotNil(t, retrievedToken.RevokedAt)

	// Verify Active Tokens is empty
	activeTokens, err = repo.GetActiveRefreshTokens(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, activeTokens, 0)
}
