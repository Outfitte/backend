package service

import (
	"errors"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

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
	err := verifyPassword("password", "!!!invalid-base64$validpart")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestVerifyPasswordShouldReturnErrUnauthorizedWhenKeyIsInvalidBase64(t *testing.T) {
	err := verifyPassword("password", "dmFsaWRzYWx0$!!!invalid-base64")
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}
