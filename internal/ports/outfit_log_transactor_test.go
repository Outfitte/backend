package ports_test

import (
	"context"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// Compile-time assertion: outfitLogTransactorStub must satisfy OutfitLogTransactor.
var _ ports.OutfitLogTransactor = (*outfitLogTransactorStub)(nil)

type outfitLogTransactorStub struct{}

func (s *outfitLogTransactorStub) CreateOutfitLog(ctx context.Context, log domain.OutfitLog, wearLogs []domain.WearLog) (domain.OutfitLog, error) {
	return domain.OutfitLog{}, nil
}

func (s *outfitLogTransactorStub) DeleteOutfitLog(ctx context.Context, outfitLogID string) error {
	return nil
}

func (s *outfitLogTransactorStub) UpdateOutfitLogDate(ctx context.Context, outfitLogID string, newDate time.Time) error {
	return nil
}
