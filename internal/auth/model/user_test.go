package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndComparePassword(t *testing.T) {
	// Use small parameters for testing to speed up execution
	testParams := Argon2Params{
		Memory:      4096, // 4 MB
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  8,
		KeyLength:   16,
	}

	password := "SecretPassword123!"

	// Hash password
	hash, err := HashPasswordWithParams(password, testParams)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify prefix and parameters
	assert.Contains(t, hash, "$argon2id$v=19$m=4096,t=1,p=1$")

	// Success comparison
	match, err := ComparePassword(password, hash)
	require.NoError(t, err)
	assert.True(t, match)

	// Wrong password comparison
	match, err = ComparePassword("WrongPassword!", hash)
	require.NoError(t, err)
	assert.False(t, match)

	// Invalid hash format error
	match, err = ComparePassword(password, "invalid_hash_string")
	assert.ErrorIs(t, err, ErrInvalidHash)
	assert.False(t, match)
}
