package domain_test

import (
	"errors"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWearLogShouldImplementPortsEntityWhenCreated(t *testing.T) {
	iface := (*ports.Entity)(nil)
	assert.Implements(t, iface, domain.WearLog{})
}
