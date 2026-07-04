package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timurdianradhasejati/telemed_hub/internal/auth/model"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrEmailConflict = errors.New("email already exists")
	ErrTokenNotFound = errors.New("refresh token not found")
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateUserWithRoleAndProfile(ctx context.Context, user *model.User, roleName string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Insert User
	queryUser := `
		INSERT INTO users (id, email, phone_number, password_hash, full_name, is_verified, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	err = tx.QueryRow(ctx, queryUser,
		user.ID, user.Email, user.PhoneNumber, user.PasswordHash, user.FullName, user.IsVerified, user.Status, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation
			return ErrEmailConflict
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}

	// 2. Fetch Role ID
	var roleID uuid.UUID
	queryRole := `SELECT id FROM roles WHERE name = $1`
	err = tx.QueryRow(ctx, queryRole, roleName).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("role %q does not exist", roleName)
		}
		return fmt.Errorf("failed to fetch role: %w", err)
	}

	// 3. Map User to Role
	queryUserRole := `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`
	_, err = tx.Exec(ctx, queryUserRole, user.ID, roleID)
	if err != nil {
		return fmt.Errorf("failed to map user to role: %w", err)
	}

	// 4. Create Profile
	if roleName == "patient" {
		queryPatient := `INSERT INTO patients (user_id, created_at, updated_at) VALUES ($1, $2, $3)`
		_, err = tx.Exec(ctx, queryPatient, user.ID, time.Now().UTC(), time.Now().UTC())
		if err != nil {
			return fmt.Errorf("failed to initialize patient profile: %w", err)
		}
	} else if roleName == "doctor" {
		queryDoctor := `INSERT INTO doctors (user_id, created_at, updated_at) VALUES ($1, $2, $3)`
		_, err = tx.Exec(ctx, queryDoctor, user.ID, time.Now().UTC(), time.Now().UTC())
		if err != nil {
			return fmt.Errorf("failed to initialize doctor profile: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, phone_number, password_hash, full_name, is_verified, status, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	var user model.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PhoneNumber, &user.PasswordHash, &user.FullName, &user.IsVerified, &user.Status,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.CreatedBy, &user.UpdatedBy, &user.DeletedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return &user, nil
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, phone_number, password_hash, full_name, is_verified, status, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	var user model.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PhoneNumber, &user.PasswordHash, &user.FullName, &user.IsVerified, &user.Status,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.CreatedBy, &user.UpdatedBy, &user.DeletedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to fetch user by ID: %w", err)
	}

	return &user, nil
}

func (r *PostgresRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT r.name
		FROM roles r
		INNER JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, fmt.Errorf("failed to scan role name: %w", err)
		}
		roles = append(roles, roleName)
	}

	return roles, nil
}

func (r *PostgresRepository) SaveRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, revoked_at, user_agent, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		token.UserID, token.TokenHash, token.ExpiresAt, token.RevokedAt, token.UserAgent, token.IPAddress, token.CreatedAt,
	).Scan(&token.ID)

	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, user_agent, ip_address, created_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	var token model.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.UserAgent, &token.IPAddress, &token.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to fetch refresh token: %w", err)
	}

	return &token, nil
}

func (r *PostgresRepository) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL`

	_, err := r.db.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

func (r *PostgresRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all user refresh tokens: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetActiveRefreshTokens(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, user_agent, ip_address, created_at
		FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active refresh tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*model.RefreshToken
	for rows.Next() {
		var token model.RefreshToken
		err := rows.Scan(
			&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.UserAgent, &token.IPAddress, &token.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token row: %w", err)
		}
		tokens = append(tokens, &token)
	}

	return tokens, nil
}
