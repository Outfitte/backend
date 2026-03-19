package json

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
)

// AppSettingsRepository is a JSON file-backed implementation of ports.AppSettingsRepository.
type AppSettingsRepository struct {
	store *SingletonStore[domain.AppSettings]
}

// NewAppSettingsRepository creates an AppSettingsRepository that stores settings in root/app_settings.json.
func NewAppSettingsRepository(root string) *AppSettingsRepository {
	return &AppSettingsRepository{
		store: NewSingletonStore[domain.AppSettings](root, "app_settings.json"),
	}
}

func (r *AppSettingsRepository) Load(ctx context.Context) (domain.AppSettings, error) {
	return r.store.Load(ctx)
}

func (r *AppSettingsRepository) Save(ctx context.Context, settings domain.AppSettings) error {
	return r.store.Save(ctx, settings)
}
