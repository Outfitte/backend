package ports_test

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// Compile-time assertion: wearLogRepositoryStub must satisfy WearLogRepository.
var _ ports.WearLogRepository = (*wearLogRepositoryStub)(nil)

type wearLogRepositoryStub struct{}

func (s *wearLogRepositoryStub) Get(ctx context.Context, _ string) (domain.WearLog, error) {
	return domain.WearLog{}, nil
}

func (s *wearLogRepositoryStub) Save(ctx context.Context, _ domain.WearLog) error {
	return nil
}

func (s *wearLogRepositoryStub) Delete(ctx context.Context, _ string) error {
	return nil
}

func (s *wearLogRepositoryStub) ListByItem(ctx context.Context, _ string) ([]domain.WearLog, error) {
	return nil, nil
}
