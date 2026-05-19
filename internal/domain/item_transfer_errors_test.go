package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/domain"
)

func TestErrItemTransferPendingShouldBeDistinctSentinelWhenComparedToOtherErrors(t *testing.T) {
	require.NotErrorIs(t, domain.ErrItemTransferPending, domain.ErrSelfTransfer)
	require.NotErrorIs(t, domain.ErrSelfTransfer, domain.ErrItemTransferPending)
}

func TestErrSelfTransferShouldBeDistinctSentinelWhenComparedToOtherErrors(t *testing.T) {
	require.NotErrorIs(t, domain.ErrSelfTransfer, domain.ErrItemTransferPending)
	require.NotErrorIs(t, domain.ErrItemTransferPending, domain.ErrSelfTransfer)
}
