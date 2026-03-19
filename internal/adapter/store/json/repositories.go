package json

import "github.com/outfitte/outfitte/internal/ports"

// NewRepositories creates a fully populated ports.Repositories backed by JSON files
// stored under dataPath. Each entity type gets its own JSON file.
func NewRepositories(dataPath string) ports.Repositories {
	return ports.Repositories{
		Items:       NewItemRepository(dataPath),
		Users:       NewUserRepository(dataPath),
		Sessions:    NewSessionRepository(dataPath),
		Locations:   NewLocationRepository(dataPath),
		WearLogs:    NewWearLogRepository(dataPath),
		AppSettings: NewAppSettingsRepository(dataPath),
	}
}
