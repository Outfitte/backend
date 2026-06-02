package domain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/domain"
)

func TestErrFutureDateNotAllowedShouldBeDistinctSentinelWhenComparedToOtherErrors(t *testing.T) {
	require.NotNil(t, domain.ErrFutureDateNotAllowed)
	require.False(t, errors.Is(domain.ErrFutureDateNotAllowed, domain.ErrValidation))
}

func TestWearLogShouldHaveNilNotesWhenNotSet(t *testing.T) {
	var wl domain.WearLog
	wl.ID = "42"
	require.Nil(t, wl.Notes)
}
