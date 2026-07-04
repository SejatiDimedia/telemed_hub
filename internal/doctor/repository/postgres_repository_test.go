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

	// Create pgx connection pool with retry loop
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

func TestPostgresRepository_DoctorOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	// 1. Insert 2 Doctor Users
	d1UserID := uuid.New()
	d1DoctorID := uuid.New()
	d2UserID := uuid.New()
	d2DoctorID := uuid.New()

	queryUser := `
		INSERT INTO users (id, email, password_hash, full_name, is_verified, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := db.Exec(ctx, queryUser, d1UserID, "amir@test.com", "hash", "Dr. Amir", true, "active", time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)
	_, err = db.Exec(ctx, queryUser, d2UserID, "budi@test.com", "hash", "Dr. Budi", true, "active", time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)

	queryDoctor := `
		INSERT INTO doctors (id, user_id, specialty, license_number, is_credential_verified, consultation_fee, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = db.Exec(ctx, queryDoctor, d1DoctorID, d1UserID, "cardiology", "123.456", false, 150000, time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)
	_, err = db.Exec(ctx, queryDoctor, d2DoctorID, d2UserID, "pediatrics", "987.654", true, 200000, time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)

	// 2. Fetch by User ID
	doc, err := repo.GetByUserID(ctx, d1UserID)
	require.NoError(t, err)
	assert.Equal(t, d1DoctorID, doc.ID)
	assert.False(t, doc.IsCredentialVerified)

	// 3. Update Profile (Transactional)
	phone := "+6281122334455"
	doc.PhoneNumber = &phone
	doc.ConsultationFee = 180000

	err = repo.Update(ctx, doc)
	require.NoError(t, err)

	updatedDoc, err := repo.GetByID(ctx, d1DoctorID)
	require.NoError(t, err)
	assert.Equal(t, "+6281122334455", *updatedDoc.PhoneNumber)
	assert.Equal(t, int64(180000), updatedDoc.ConsultationFee)

	// 4. Verify Credentials
	err = repo.Verify(ctx, d1DoctorID)
	require.NoError(t, err)

	verifiedDoc, err := repo.GetByID(ctx, d1DoctorID)
	require.NoError(t, err)
	assert.True(t, verifiedDoc.IsCredentialVerified)

	// 5. List with filters & pagination
	// Scenario A: Unverified + verified (Admin sees all)
	docs, total, err := repo.List(ctx, nil, false, "consultation_fee", "desc", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, docs, 2)
	assert.Equal(t, "Dr. Budi", docs[0].FullName) // Fee 200000 > 180000 (sorted descending)

	// Scenario B: Filter specialty
	spec := "cardiology"
	docsSpec, totalSpec, err := repo.List(ctx, &spec, false, "created_at", "desc", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, totalSpec)
	assert.Equal(t, "Dr. Amir", docsSpec[0].FullName)
}
