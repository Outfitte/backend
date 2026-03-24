package domain_test

import (
	"testing"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestOutfitLogShouldHaveNilNotesWhenNotSet(t *testing.T) {
	var ol domain.OutfitLog
	ol.ID = "42"
	ol.OutfitID = "outfit-1"
	ol.OwnerID = "owner-1"
	ol.WornOn = time.Now()
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
	assert.Equal(t, "casual day", *ol.Notes)
}

func TestOutfitLogShouldHoldWearLogIDsWhenPopulated(t *testing.T) {
	var ol domain.OutfitLog
	ol.ID = "42"
	ol.WearLogIDs = []string{"wl-1", "wl-2"}
	assert.Len(t, ol.WearLogIDs, 2)
	assert.Equal(t, "wl-1", ol.WearLogIDs[0])
}
