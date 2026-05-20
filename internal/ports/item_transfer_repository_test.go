package ports_test

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// Compile-time assertion: itemTransferRepositoryStub must satisfy ItemTransferRepository.
var _ ports.ItemTransferRepository = (*itemTransferRepositoryStub)(nil)

type itemTransferRepositoryStub struct{}

func (s *itemTransferRepositoryStub) Get(ctx context.Context, _ string) (domain.ItemTransfer, error) {
	return domain.ItemTransfer{}, nil
}

func (s *itemTransferRepositoryStub) Save(ctx context.Context, _ domain.ItemTransfer) error {
	return nil
}

func (s *itemTransferRepositoryStub) Delete(ctx context.Context, _ string) error {
	return nil
}

func (s *itemTransferRepositoryStub) ListBySender(ctx context.Context, _ string, _ *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	return nil, nil
}

func (s *itemTransferRepositoryStub) ListByRecipient(ctx context.Context, _ string, _ *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	return nil, nil
}

func (s *itemTransferRepositoryStub) FindPendingByItem(ctx context.Context, _ string) (*domain.ItemTransfer, error) {
	return nil, nil
}

func (s *itemTransferRepositoryStub) HasPending(ctx context.Context, _ string) (bool, error) {
	return false, nil
}
