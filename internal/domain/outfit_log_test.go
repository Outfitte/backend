package domain_test

import (
	"testing"

	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutfitLogShouldHaveNilNotesWhenNotSet(t *testing.T) {
	var ol domain.OutfitLog
	ol.ID = "42"
	assert.Nil(t, ol.Notes)
}

func TestOutfitLogShouldHaveEmptyWearLogIDsWhenNotPopulated(t *testing.T) {
	var ol domain.OutfitLog
	ol.ID = "42"
	assert.Empty(t, ol.WearLogIDs)
}

func TestOutfitLogShouldHoldNotesWhenSet(t *testing.T) {
	var ol domain.OutfitLog
	ol.ID = "42"
	notes := "casual day"
	ol.Notes = &notes
	require.NotNil(t, ol.Notes)
	assert.Equal(t, "casual day", *ol.Notes)
}

func TestOutfitLogShouldHoldWearLogIDsWhenPopulated(t *testing.T) {
	var ol domain.OutfitLog
	ol.ID = "42"
	ol.WearLogIDs = []string{"wl-1", "wl-2"}
	require.Len(t, ol.WearLogIDs, 2)
	assert.Equal(t, "wl-1", ol.WearLogIDs[0])
}
