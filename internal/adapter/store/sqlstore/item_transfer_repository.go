package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// itemTransferDB is the subset of *sql.DB methods used by ItemTransferRepository.
type itemTransferDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ ports.ItemTransferRepository = (*ItemTransferRepository)(nil)

// ItemTransferRepository is a SQL-backed implementation of ports.ItemTransferRepository.
type ItemTransferRepository struct {
	db itemTransferDB
}

// NewItemTransferRepository creates an ItemTransferRepository backed by the given db.
func NewItemTransferRepository(db itemTransferDB) *ItemTransferRepository {
	return &ItemTransferRepository{db: db}
}

func (r *ItemTransferRepository) Get(_ context.Context, _ string) (domain.ItemTransfer, error) {
	return domain.ItemTransfer{}, errors.New("not implemented")
}

func (r *ItemTransferRepository) Save(_ context.Context, _ domain.ItemTransfer) error {
	return errors.New("not implemented")
}

func (r *ItemTransferRepository) Delete(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

func (r *ItemTransferRepository) ListBySender(_ context.Context, _ string, _ *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	return nil, errors.New("not implemented")
}

func (r *ItemTransferRepository) ListByRecipient(_ context.Context, _ string, _ *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	return nil, errors.New("not implemented")
}

func (r *ItemTransferRepository) FindPendingByItem(_ context.Context, _ string) (*domain.ItemTransfer, error) {
	return nil, errors.New("not implemented")
}

func (r *ItemTransferRepository) HasPending(_ context.Context, _ string) (bool, error) {
	return false, errors.New("not implemented")
}
