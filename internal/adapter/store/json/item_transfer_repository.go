package json

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

var _ ports.ItemTransferRepository = (*ItemTransferRepository)(nil)

// itemTransferProvider is the subset of Provider[domain.ItemTransfer] methods used by ItemTransferRepository.
type itemTransferProvider interface {
	Get(ctx context.Context, id string) (domain.ItemTransfer, error)
	Save(ctx context.Context, transfer domain.ItemTransfer) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]domain.ItemTransfer, error)
}

// ItemTransferRepository is a JSON file-backed implementation of ports.ItemTransferRepository.
type ItemTransferRepository struct {
	provider itemTransferProvider
}

// NewItemTransferRepository creates an ItemTransferRepository that stores transfers in root/item_transfers.json.
func NewItemTransferRepository(root string) *ItemTransferRepository {
	return &ItemTransferRepository{
		provider: NewProvider[domain.ItemTransfer](root, "item_transfers.json"),
	}
}

func (r *ItemTransferRepository) Get(ctx context.Context, id string) (domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return domain.ItemTransfer{}, err
	}
	return r.provider.Get(ctx, id)
}

func (r *ItemTransferRepository) Save(ctx context.Context, transfer domain.ItemTransfer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.provider.Save(ctx, transfer)
}

func (r *ItemTransferRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.provider.Delete(ctx, id)
}

func (r *ItemTransferRepository) ListBySender(ctx context.Context, senderID string, statusFilter *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []domain.ItemTransfer{}
	for _, tr := range all {
		if tr.SenderID == senderID && (statusFilter == nil || tr.Status == *statusFilter) {
			result = append(result, tr)
		}
	}
	return result, nil
}

func (r *ItemTransferRepository) ListByRecipient(ctx context.Context, recipientID string, statusFilter *domain.TransferStatus) ([]domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []domain.ItemTransfer{}
	for _, tr := range all {
		if tr.RecipientID == recipientID && (statusFilter == nil || tr.Status == *statusFilter) {
			result = append(result, tr)
		}
	}
	return result, nil
}

func (r *ItemTransferRepository) FindPendingByItem(ctx context.Context, itemID string) (*domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, tr := range all {
		if tr.ItemID == itemID && tr.Status == domain.TransferStatusPending {
			found := tr
			return &found, nil
		}
	}
	return nil, nil
}

func (r *ItemTransferRepository) HasPending(ctx context.Context, itemID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	tr, err := r.FindPendingByItem(ctx, itemID)
	if err != nil {
		return false, err
	}
	return tr != nil, nil
}
