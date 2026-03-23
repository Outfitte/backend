package sqlstore

import (
	"database/sql"

	"github.com/outfitte/outfitte/internal/ports"
)

// NewRepositories creates a fully populated ports.Repositories backed by the given SQL database.
func NewRepositories(db *sql.DB) ports.Repositories {
	return ports.Repositories{
		Items:       NewItemRepository(db),
		Users:       NewUserRepository(db),
		Sessions:    NewSessionRepository(db),
		Locations:   NewLocationRepository(db),
		WearLogs:    NewWearLogRepository(db),
		AppSettings: NewAppSettingsRepository(db),
	}
}
