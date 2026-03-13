package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashTokenShouldReturn64HexCharsWhenGivenInput(t *testing.T) {
	result := hashToken([]byte("secret"), "randompart")
	require.Len(t, result, 64) // HMAC-SHA256 = 32 bytes = 64 hex chars
}

func TestHashTokenShouldBeDeterministicWhenGivenSameInputs(t *testing.T) {
	h1 := hashToken([]byte("secret"), "randompart")
	h2 := hashToken([]byte("secret"), "randompart")
	require.Equal(t, h1, h2)
}

func TestHashTokenShouldDifferWhenSecretDiffers(t *testing.T) {
	h1 := hashToken([]byte("secret-a"), "randompart")
	h2 := hashToken([]byte("secret-b"), "randompart")
	require.NotEqual(t, h1, h2)
}

func TestHashTokenShouldDifferWhenRawRandomDiffers(t *testing.T) {
	secret := []byte("secret")
	h1 := hashToken(secret, "randompart-a")
	h2 := hashToken(secret, "randompart-b")
	require.NotEqual(t, h1, h2)
}
