package service

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestHashPasswordShouldWrapErrIOWhenRandReadFails(t *testing.T) {
	failingRand := func(b []byte) (int, error) {
		return 0, errors.New("entropy failure")
	}
	_, err := hashPassword("password", failingRand)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestHashPasswordShouldProducePHCFormatWhenSuccessful(t *testing.T) {
	hash, err := hashPassword("password", rand.Read)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(hash, "$argon2id$v=19$"), "expected PHC prefix, got: %s", hash)
}

func TestVerifyPasswordShouldReturnErrUnauthorizedWhenHashHasNoSeparator(t *testing.T) {
	err := verifyPassword("password", "noDollarSignInThisHash")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestVerifyPasswordShouldReturnErrUnauthorizedWhenAlgoIsNotArgon2id(t *testing.T) {
	// Generate a valid hash then swap the algo field — key is correct for "password"
	// so without algo validation the function would return nil.
	hash, err := hashPassword("password", rand.Read)
	require.NoError(t, err)
	tampered := strings.Replace(hash, "$argon2id$", "$bcrypt$", 1)
	err = verifyPassword("password", tampered)
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestVerifyPasswordShouldReturnErrUnauthorizedWhenVersionIsWrong(t *testing.T) {
	// Generate a valid hash then swap the version field.
	hash, err := hashPassword("password", rand.Read)
	require.NoError(t, err)
	tampered := strings.Replace(hash, "$v=19$", "$v=18$", 1)
	err = verifyPassword("password", tampered)
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestVerifyPasswordShouldReturnErrUnauthorizedWhenParamsAreMalformed(t *testing.T) {
	err := verifyPassword("password", "$argon2id$v=19$invalid-params$dmFsaWRzYWx0$dmFsaWRrZXk")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestVerifyPasswordShouldReturnErrUnauthorizedWhenSaltIsInvalidBase64(t *testing.T) {
	err := verifyPassword("password", "$argon2id$v=19$m=65536,t=3,p=2$!!!invalid-base64$dmFsaWRzYWx0")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestVerifyPasswordShouldReturnErrUnauthorizedWhenKeyIsInvalidBase64(t *testing.T) {
	err := verifyPassword("password", "$argon2id$v=19$m=65536,t=3,p=2$dmFsaWRzYWx0$!!!invalid-base64")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestVerifyPasswordShouldReturnErrUnauthorizedWhenPasswordIsWrong(t *testing.T) {
	hash, err := hashPassword("correct-password", rand.Read)
	require.NoError(t, err)

	err = verifyPassword("wrong-password", hash)
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestVerifyPasswordShouldReturnNilWhenPasswordMatches(t *testing.T) {
	hash, err := hashPassword("correct-password", rand.Read)
	require.NoError(t, err)

	err = verifyPassword("correct-password", hash)
	require.NoError(t, err)
}
