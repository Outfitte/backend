package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashPasswordShouldReturnErrWhenRandReadFails(t *testing.T) {
	failingRand := func(b []byte) (int, error) {
		return 0, errors.New("entropy failure")
	}
	_, err := hashPassword("password", failingRand)
	require.Error(t, err)
}
