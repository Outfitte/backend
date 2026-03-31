package ports_test

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// Compile-time assertion: outfitRepositoryStub must satisfy OutfitRepository.
var _ ports.OutfitRepository = (*outfitRepositoryStub)(nil)

type outfitRepositoryStub struct{}

func (s *outfitRepositoryStub) Get(_ context.Context, _ string) (domain.Outfit, error) {
	return domain.Outfit{}, nil
}

func (s *outfitRepositoryStub) Save(_ context.Context, _ domain.Outfit) error {
	return nil
}

func (s *outfitRepositoryStub) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *outfitRepositoryStub) ListByOwner(_ context.Context, _ string) ([]domain.Outfit, error) {
	return nil, nil
}

func (s *outfitRepositoryStub) SaveItem(_ context.Context, _, _ string, _ int) error {
	return nil
}

func (s *outfitRepositoryStub) DeleteItem(_ context.Context, _, _ string) error {
	return nil
}

func (s *outfitRepositoryStub) ListItemIDs(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (s *outfitRepositoryStub) SavePhoto(_ context.Context, _, _, _ string, _ int) error {
	return nil
}

func (s *outfitRepositoryStub) DeletePhoto(_ context.Context, _, _ string) error {
	return nil
}
