package ports_test

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// Compile-time assertion: locationRepositoryStub must satisfy LocationRepository.
var _ ports.LocationRepository = (*locationRepositoryStub)(nil)

type locationRepositoryStub struct{}

func (s *locationRepositoryStub) Get(ctx context.Context, _ string) (domain.Location, error) {
	return domain.Location{}, nil
}

func (s *locationRepositoryStub) Save(ctx context.Context, _ domain.Location) error {
	return nil
}

func (s *locationRepositoryStub) Delete(ctx context.Context, _ string) error {
	return nil
}

func (s *locationRepositoryStub) ListByOwner(ctx context.Context, _ string) ([]domain.Location, error) {
	return nil, nil
}

func (s *locationRepositoryStub) HasChildren(ctx context.Context, _ string) (bool, error) {
	return false, nil
}
