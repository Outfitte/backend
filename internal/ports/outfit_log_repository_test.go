package ports_test

import (
	"context"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// Compile-time assertion: outfitLogRepositoryStub must satisfy OutfitLogRepository.
var _ ports.OutfitLogRepository = (*outfitLogRepositoryStub)(nil)

type outfitLogRepositoryStub struct{}

func (s *outfitLogRepositoryStub) Get(ctx context.Context, _ string) (domain.OutfitLog, error) {
	return domain.OutfitLog{}, nil
}

func (s *outfitLogRepositoryStub) Save(ctx context.Context, _ domain.OutfitLog) error {
	return nil
}

func (s *outfitLogRepositoryStub) Delete(ctx context.Context, _ string) error {
	return nil
}

func (s *outfitLogRepositoryStub) ListByOutfit(ctx context.Context, _ string) ([]domain.OutfitLog, error) {
	return nil, nil
}

func (s *outfitLogRepositoryStub) ListByOwnerDateRange(ctx context.Context, _ string, _, _ time.Time) ([]domain.OutfitLog, error) {
	return nil, nil
}

func (s *outfitLogRepositoryStub) LinkWearLog(ctx context.Context, _, _ string) error {
	return nil
}

func (s *outfitLogRepositoryStub) LinkedWearLogIDs(ctx context.Context, _ string) ([]string, error) {
	return nil, nil
}
