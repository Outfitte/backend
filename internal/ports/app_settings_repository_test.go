package ports_test

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// Compile-time assertion: appSettingsRepositoryStub must satisfy AppSettingsRepository.
var _ ports.AppSettingsRepository = (*appSettingsRepositoryStub)(nil)

type appSettingsRepositoryStub struct{}

func (s *appSettingsRepositoryStub) Load(ctx context.Context) (domain.AppSettings, error) {
	return domain.AppSettings{}, nil
}

func (s *appSettingsRepositoryStub) Save(ctx context.Context, _ domain.AppSettings) error {
	return nil
}
