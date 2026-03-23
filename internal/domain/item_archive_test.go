package domain_test

import (
	"testing"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Error sentinels ───────────────────────────────────────────────────────────

func TestErrAlreadyArchivedShouldBeDistinctSentinelWhenComparedToOtherErrors(t *testing.T) {
	require.NotNil(t, domain.ErrAlreadyArchived)
	require.NotEqual(t, domain.ErrAlreadyArchived, domain.ErrNotArchived)
}

func TestErrNotArchivedShouldBeDistinctSentinelWhenComparedToOtherErrors(t *testing.T) {
	require.NotNil(t, domain.ErrNotArchived)
	require.NotEqual(t, domain.ErrNotArchived, domain.ErrAlreadyArchived)
}

// ── DisposalReason constants ──────────────────────────────────────────────────

func TestDisposalReasonConstantsShouldHaveExpectedValuesWhenDefined(t *testing.T) {
	assert.Equal(t, domain.DisposalReason("donated"), domain.DisposalDonated)
	assert.Equal(t, domain.DisposalReason("sold"), domain.DisposalSold)
	assert.Equal(t, domain.DisposalReason("discarded"), domain.DisposalDiscarded)
	assert.Equal(t, domain.DisposalReason("lost"), domain.DisposalLost)
	assert.Equal(t, domain.DisposalReason("other"), domain.DisposalOther)
}

func TestDisposalReasonConstantsShouldBeDistinctWhenCompared(t *testing.T) {
	reasons := []domain.DisposalReason{
		domain.DisposalDonated,
		domain.DisposalSold,
		domain.DisposalDiscarded,
		domain.DisposalLost,
		domain.DisposalOther,
	}
	seen := make(map[domain.DisposalReason]bool)
	for _, r := range reasons {
		assert.False(t, seen[r], "duplicate DisposalReason constant: %s", r)
		seen[r] = true
	}
}

// ── Item.ArchivedAt ───────────────────────────────────────────────────────────

func TestItemArchivedAtShouldBeNilWhenNotArchived(t *testing.T) {
	var item domain.Item
	item.ID = "42"
	assert.Nil(t, item.ArchivedAt)
}

func TestItemShouldBeConsideredArchivedWhenArchivedAtIsNonNil(t *testing.T) {
	var item domain.Item
	item.ID = "42"
	now := time.Now()
	item.ArchivedAt = &now
	assert.NotNil(t, item.ArchivedAt)
}

// ── Item.DisposalReason ───────────────────────────────────────────────────────

func TestItemDisposalReasonShouldBeNilWhenNotDisposed(t *testing.T) {
	var item domain.Item
	item.ID = "42"
	assert.Nil(t, item.DisposalReason)
}

func TestItemShouldBeConsideredDisposedWhenDisposalReasonIsNonNil(t *testing.T) {
	var item domain.Item
	item.ID = "42"
	reason := domain.DisposalDonated
	item.DisposalReason = &reason
	assert.NotNil(t, item.DisposalReason)
	assert.Equal(t, domain.DisposalDonated, *item.DisposalReason)
}

func TestItemFieldsShouldRepresentDisposedStateWhenBothAreSet(t *testing.T) {
	var item domain.Item
	item.ID = "42"
	now := time.Now()
	reason := domain.DisposalSold
	item.ArchivedAt = &now
	item.DisposalReason = &reason
	assert.NotNil(t, item.ArchivedAt)
	assert.NotNil(t, item.DisposalReason)
}
