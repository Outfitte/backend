package domain_test

import (
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// ── ValidateMetadataKey ───────────────────────────────────────────────────────

func TestValidateMetadataKeyShouldReturnErrValidationWhenKeyIsEmpty(t *testing.T) {
	err := domain.ValidateMetadataKey("")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidateMetadataKeyShouldReturnErrValidationWhenKeyExceedsMaxLength(t *testing.T) {
	key := strings.Repeat("a", 65)
	err := domain.ValidateMetadataKey(key)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidateMetadataKeyShouldReturnErrValidationWhenKeyContainsSpecialChars(t *testing.T) {
	err := domain.ValidateMetadataKey("size!")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidateMetadataKeyShouldReturnErrValidationWhenKeyHasLeadingSpace(t *testing.T) {
	err := domain.ValidateMetadataKey(" size")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidateMetadataKeyShouldReturnErrValidationWhenKeyHasTrailingSpace(t *testing.T) {
	err := domain.ValidateMetadataKey("size ")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidateMetadataKeyShouldReturnNilWhenKeyIsValid(t *testing.T) {
	err := domain.ValidateMetadataKey("shoe size")
	require.NoError(t, err)
}

func TestValidateMetadataKeyShouldReturnNilWhenKeyIsExactlyMaxLength(t *testing.T) {
	key := strings.Repeat("a", 64)
	err := domain.ValidateMetadataKey(key)
	require.NoError(t, err)
}

// ── ValidateMetadata ──────────────────────────────────────────────────────────

func TestValidateMetadataShouldReturnErrValidationWhenFieldCountExceedsMax(t *testing.T) {
	m := domain.ItemMetadata{Fields: make(map[string]string, 51)}
	for i := range 51 {
		m.Fields["field"+string(rune('a'+i%26))+string(rune('0'+i/26))] = "v"
	}
	err := domain.ValidateMetadata(m)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidateMetadataShouldReturnErrValidationWhenValueExceedsMaxLength(t *testing.T) {
	m := domain.ItemMetadata{Fields: map[string]string{
		"size": strings.Repeat("x", 513),
	}}
	err := domain.ValidateMetadata(m)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidateMetadataShouldReturnErrValidationWhenKeyIsInvalid(t *testing.T) {
	m := domain.ItemMetadata{Fields: map[string]string{
		"bad!key": "value",
	}}
	err := domain.ValidateMetadata(m)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestValidateMetadataShouldReturnNilWhenMetadataIsValid(t *testing.T) {
	m := domain.ItemMetadata{Fields: map[string]string{
		"size":   "M",
		"color":  "blue",
		"fit":    "slim",
		"season": "winter",
	}}
	err := domain.ValidateMetadata(m)
	require.NoError(t, err)
}

func TestValidateMetadataShouldReturnNilWhenFieldsIsNil(t *testing.T) {
	m := domain.ItemMetadata{}
	err := domain.ValidateMetadata(m)
	require.NoError(t, err)
}

func TestValidateMetadataShouldReturnNilWhenValueIsExactlyMaxLength(t *testing.T) {
	m := domain.ItemMetadata{Fields: map[string]string{
		"size": strings.Repeat("x", 512),
	}}
	err := domain.ValidateMetadata(m)
	require.NoError(t, err)
}

func TestValidateMetadataShouldReturnNilWhenFieldCountIsExactlyMax(t *testing.T) {
	m := domain.ItemMetadata{Fields: make(map[string]string, 50)}
	for i := range 50 {
		m.Fields["field"+string(rune('a'+i%26))+string(rune('0'+i/26))] = "v"
	}
	err := domain.ValidateMetadata(m)
	require.NoError(t, err)
}
