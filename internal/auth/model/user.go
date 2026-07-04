package model

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("the encoded password hash has an invalid format")
	ErrIncompatibleVersion = errors.New("the password hash has an incompatible version")
)

// User represents the central identity of a patient, doctor, pharmacy staff, or admin.
type User struct {
	ID           uuid.UUID
	Email        string
	PhoneNumber  *string
	PasswordHash string
	FullName     string
	IsVerified   bool
	Status       string // "active", "suspended", "deactivated"
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	CreatedBy    *uuid.UUID
	UpdatedBy    *uuid.UUID
	DeletedBy    *uuid.UUID
}

// Role represents a system role (e.g. patient, doctor, admin, pharmacy_staff).
type Role struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRole links a user to a role.
type UserRole struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	RoleID    uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RefreshToken represents a stateful session token stored in the database.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	UserAgent *string
	IPAddress *string
	CreatedAt time.Time
}

// Argon2Params defines parameters for Argon2id hashing.
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams are the recommended parameters from the auth specification doc.
var DefaultParams = Argon2Params{
	Memory:      65536, // 64 MB
	Iterations:  1,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

// HashPassword hashes a password using Argon2id with default parameters.
func HashPassword(password string) (string, error) {
	return HashPasswordWithParams(password, DefaultParams)
}

// HashPasswordWithParams hashes a password with the provided custom parameters.
func HashPasswordWithParams(password string, params Argon2Params) (string, error) {
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate random salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$<salt_base64>$<hash_base64>
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, params.Memory, params.Iterations, params.Parallelism, saltB64, hashB64)

	return encoded, nil
}

// ComparePassword compares a plaintext password with an Argon2id hash.
// Timing-safe to prevent side-channel attacks.
func ComparePassword(password, encodedHash string) (bool, error) {
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Use subtle.ConstantTimeCompare to avoid timing side-channels
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

func decodeHash(encodedHash string) (params Argon2Params, salt, hash []byte, err error) {
	parts := strings.Split(encodedHash, "$")
	// Expected parts: ["", "argon2id", "v=19", "m=65536,t=1,p=4", "<salt>", "<hash>"]
	if len(parts) != 6 {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}

	var version int
	_, err = fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return Argon2Params{}, nil, nil, err
	}
	if version != argon2.Version {
		return Argon2Params{}, nil, nil, ErrIncompatibleVersion
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism)
	if err != nil {
		return Argon2Params{}, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, err
	}
	params.SaltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, err
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}
