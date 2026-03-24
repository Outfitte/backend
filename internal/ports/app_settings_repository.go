package ports

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
)

// AppSettingsRepository is the storage port for AppSettings persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type AppSettingsRepository interface {
	// Load retrieves the singleton application settings.
	// Returns domain.ErrNotFound if no settings have been saved yet.
	Load(ctx context.Context) (domain.AppSettings, error)

	// Save creates or updates the singleton application settings.
	Save(ctx context.Context, settings domain.AppSettings) error
}
