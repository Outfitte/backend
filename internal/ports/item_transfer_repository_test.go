package ports_test

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// Compile-time assertion: itemTransferRepositoryStub must satisfy ItemTransferRepository.
var _ ports.ItemTransferRepository = (*itemTransferRepositoryStub)(nil)

type itemTransferRepositoryStub struct{}

func (s *itemTransferRepositoryStub) Get(_ context.Context, _ string) (domain.ItemTransfer, error) {
	return domain.ItemTransfer{}, nil
}

func (s *itemTransferRepositoryStub) Save(_ context.Context, _ domain.ItemTransfer) error {
	return nil
}

func (s *itemTransferRepositoryStub) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *itemTransferRepositoryStub) ListBySender(_ context.Context, _ string, _ *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	return nil, nil
}

func (s *itemTransferRepositoryStub) ListByRecipient(_ context.Context, _ string, _ *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	return nil, nil
}

func (s *itemTransferRepositoryStub) FindPendingByItem(_ context.Context, _ string) (*domain.ItemTransfer, error) {
	return nil, nil
}

func (s *itemTransferRepositoryStub) HasPending(_ context.Context, _ string) (bool, error) {
	return false, nil
}
