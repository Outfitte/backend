package ports_test

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// Compile-time assertion: wearLogRepositoryStub must satisfy WearLogRepository.
var _ ports.WearLogRepository = (*wearLogRepositoryStub)(nil)

type wearLogRepositoryStub struct{}

func (s *wearLogRepositoryStub) Get(ctx context.Context, id string) (domain.WearLog, error) {
	return domain.WearLog{}, nil
}

func (s *wearLogRepositoryStub) Save(ctx context.Context, log domain.WearLog) error {
	return nil
}

func (s *wearLogRepositoryStub) Delete(ctx context.Context, id string) error {
	return nil
}

func (s *wearLogRepositoryStub) ListByItem(ctx context.Context, itemID string) ([]domain.WearLog, error) {
	return nil, nil
}

func (s *wearLogRepositoryStub) LatestByItem(ctx context.Context, itemID string) (*domain.WearLog, error) {
	return nil, nil
}

func (s *wearLogRepositoryStub) CountByItem(ctx context.Context, itemID string) (int, error) {
	return 0, nil
}
