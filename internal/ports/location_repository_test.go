package ports_test

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// Compile-time assertion: locationRepositoryStub must satisfy LocationRepository.
var _ ports.LocationRepository = (*locationRepositoryStub)(nil)

type locationRepositoryStub struct{}

func (s *locationRepositoryStub) Get(_ context.Context, _ string) (domain.Location, error) {
	return domain.Location{}, nil
}

func (s *locationRepositoryStub) Save(_ context.Context, _ domain.Location) error {
	return nil
}

func (s *locationRepositoryStub) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *locationRepositoryStub) ListByOwner(_ context.Context, _ string) ([]domain.Location, error) {
	return nil, nil
}

func (s *locationRepositoryStub) HasChildren(_ context.Context, _ string) (bool, error) {
	return false, nil
}
