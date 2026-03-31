package ports_test

import (
	"context"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// Compile-time assertion: outfitLogTransactorStub must satisfy OutfitLogTransactor.
var _ ports.OutfitLogTransactor = (*outfitLogTransactorStub)(nil)

type outfitLogTransactorStub struct{}

func (s *outfitLogTransactorStub) CreateOutfitLog(ctx context.Context, _ domain.OutfitLog, _ []domain.WearLog) (domain.OutfitLog, error) {
	return domain.OutfitLog{}, nil
}

func (s *outfitLogTransactorStub) DeleteOutfitLog(ctx context.Context, _ string) error {
	return nil
}

func (s *outfitLogTransactorStub) UpdateOutfitLogDate(ctx context.Context, _ string, _ time.Time) error {
	return nil
}
