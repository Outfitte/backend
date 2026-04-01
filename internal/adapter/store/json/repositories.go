package json

import "github.com/outfitte/backend/internal/ports"

// NewRepositories creates a fully populated ports.Repositories backed by JSON files
// stored under dataPath. Each entity type gets its own JSON file.
func NewRepositories(dataPath string) ports.Repositories {
	outfitLogs := NewOutfitLogRepository(dataPath)
	wearLogs := NewWearLogRepository(dataPath)
	return ports.Repositories{
		Items:               NewItemRepository(dataPath),
		Users:               NewUserRepository(dataPath),
		Sessions:            NewSessionRepository(dataPath),
		Locations:           NewLocationRepository(dataPath),
		WearLogs:            wearLogs,
		AppSettings:         NewAppSettingsRepository(dataPath),
		Outfits:             NewOutfitRepository(dataPath),
		OutfitLogs:          outfitLogs,
		OutfitLogTransactor: NewOutfitLogTransactor(outfitLogs, wearLogs),
		Shares:              NewShareRepository(dataPath),
	}
}
