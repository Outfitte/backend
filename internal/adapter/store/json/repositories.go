package json

import (
	"sync"

	"github.com/outfitte/backend/internal/ports"
)

// NewRepositories creates a fully populated ports.Repositories backed by JSON files
// stored under dataPath. Each entity type gets its own JSON file.
func NewRepositories(dataPath string) ports.Repositories {
	outfitLogs := NewOutfitLogRepository(dataPath)
	wearLogs := NewWearLogRepository(dataPath)
	items := NewItemRepository(dataPath)
	outfits := NewOutfitRepository(dataPath)
	shares := NewShareRepository(dataPath)
	itemTransfers := NewItemTransferRepository(dataPath)
	mu := &sync.Mutex{}
	return ports.Repositories{
		Items:               items,
		Users:               NewUserRepository(dataPath),
		Sessions:            NewSessionRepository(dataPath),
		Locations:           NewLocationRepository(dataPath),
		WearLogs:            wearLogs,
		AppSettings:         NewAppSettingsRepository(dataPath),
		Outfits:             outfits,
		OutfitLogs:          outfitLogs,
		OutfitLogTransactor: NewOutfitLogTransactor(outfitLogs, wearLogs),
		Shares:              shares,
		ItemTransfers:       itemTransfers,
		ItemTransferTransactor: NewItemTransferTransactor(itemTransfers, items, wearLogs, outfits, shares, outfitLogs, mu),
	}
}
