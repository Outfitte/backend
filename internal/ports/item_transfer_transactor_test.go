package ports_test

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// Compile-time assertion: itemTransferTransactorStub must satisfy ItemTransferTransactor.
var _ ports.ItemTransferTransactor = (*itemTransferTransactorStub)(nil)

type itemTransferTransactorStub struct{}

func (s *itemTransferTransactorStub) Accept(ctx context.Context, _ string) (domain.ItemTransfer, error) {
	return domain.ItemTransfer{}, nil
}
