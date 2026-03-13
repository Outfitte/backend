package service

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestHashPasswordShouldProducePHCFormatWhenSuccessful(t *testing.T) {
	hash, err := hashPassword("password", rand.Read)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(hash, "$argon2id$v=19$"), "expected PHC prefix, got: %s", hash)
}

func TestHashPasswordShouldReturnErrWhenRandReadFails(t *testing.T) {
	failingRand := func(b []byte) (int, error) {
		return 0, errors.New("entropy failure")
	}
	_, err := hashPassword("password", failingRand)
	require.Error(t, err)
}

func TestVerifyPasswordShouldReturnErrUnauthorizedWhenHashHasNoSeparator(t *testing.T) {
	err := verifyPassword("password", "noDollarSignInThisHash")
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
