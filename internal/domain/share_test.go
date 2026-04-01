package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

func TestErrSelfShareShouldBeDistinctFromErrDuplicateShareWhenCompared(t *testing.T) {
	require.NotErrorIs(t, domain.ErrSelfShare, domain.ErrDuplicateShare)
	require.NotErrorIs(t, domain.ErrDuplicateShare, domain.ErrSelfShare)
}

func TestShareShouldHoldAllFieldsWhenConstructed(t *testing.T) {
	now := time.Now()
	var s domain.Share
	s.ID = "share-1"
	s.OwnerID = "owner-1"
	s.RecipientID = "recipient-1"
	s.TargetType = domain.ShareTargetItem
	s.TargetID = "item-1"
	s.CreatedAt = now

	assert.Equal(t, "share-1", s.ID)
	assert.Equal(t, "owner-1", s.OwnerID)
	assert.Equal(t, "recipient-1", s.RecipientID)
	assert.Equal(t, domain.ShareTargetItem, s.TargetType)
	assert.Equal(t, "item-1", s.TargetID)
	assert.Equal(t, now, s.CreatedAt)
}

func TestShareShouldImplementPortsEntityWhenGetIDCalled(t *testing.T) {
	iface := (*ports.Entity)(nil)
	assert.Implements(t, iface, domain.Share{})

	var s domain.Share
	s.ID = "share-99"
	assert.Equal(t, "share-99", s.GetID())
}

func TestShareTargetTypeShouldMatchConstantsWhenUsed(t *testing.T) {
	assert.Equal(t, domain.ShareTargetType("item"), domain.ShareTargetItem)
	assert.Equal(t, domain.ShareTargetType("outfit"), domain.ShareTargetOutfit)
	assert.Equal(t, domain.ShareTargetType("location"), domain.ShareTargetLocation)
}
