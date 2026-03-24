package ports_test

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// Compile-time assertion: outfitRepositoryStub must satisfy OutfitRepository.
var _ ports.OutfitRepository = (*outfitRepositoryStub)(nil)

type outfitRepositoryStub struct{}

func (s *outfitRepositoryStub) Get(ctx context.Context, _ string) (domain.Outfit, error) {
	return domain.Outfit{}, nil
}

func (s *outfitRepositoryStub) Save(ctx context.Context, _ domain.Outfit) error {
	return nil
}

func (s *outfitRepositoryStub) Delete(ctx context.Context, _ string) error {
	return nil
}

func (s *outfitRepositoryStub) ListByOwner(ctx context.Context, _ string) ([]domain.Outfit, error) {
	return nil, nil
}

func (s *outfitRepositoryStub) SaveItem(ctx context.Context, _, _ string, _ int) error {
	return nil
}

func (s *outfitRepositoryStub) DeleteItem(ctx context.Context, _, _ string) error {
	return nil
}

func (s *outfitRepositoryStub) ListItemIDs(ctx context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (s *outfitRepositoryStub) SavePhoto(ctx context.Context, _, _, _ string, _ int) error {
	return nil
}

func (s *outfitRepositoryStub) DeletePhoto(ctx context.Context, _, _ string) error {
	return nil
}
