package json

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

var _ ports.ItemTransferTransactor = (*ItemTransferTransactor)(nil)

// ItemTransferTransactor is a JSON file-backed implementation of ports.ItemTransferTransactor.
// Atomicity is achieved via a shared in-process mutex that serializes concurrent Accept calls
// against each other.
//
// NOTE: This implementation does not provide true atomicity — if a step fails partway through,
// prior steps are not rolled back. This is an accepted limitation for single-instance personal use.
// The mutex guards only ItemTransferTransactor calls; it is not shared with OutfitLogTransactor.
type ItemTransferTransactor struct {
	transfers  ports.ItemTransferRepository
	items      ports.ItemRepository
	wearLogs   ports.WearLogRepository
	outfits    ports.OutfitRepository
	shares     ports.ShareRepository
	outfitLogs ports.OutfitLogRepository
	mu         *sync.Mutex
}

// NewItemTransferTransactor creates an ItemTransferTransactor using the given repositories and
// shared mutex.
func NewItemTransferTransactor(
	transfers ports.ItemTransferRepository,
	items ports.ItemRepository,
	wearLogs ports.WearLogRepository,
	outfits ports.OutfitRepository,
	shares ports.ShareRepository,
	outfitLogs ports.OutfitLogRepository,
	mu *sync.Mutex,
) *ItemTransferTransactor {
	return &ItemTransferTransactor{
		transfers:  transfers,
		items:      items,
		wearLogs:   wearLogs,
		outfits:    outfits,
		shares:     shares,
		outfitLogs: outfitLogs,
		mu:         mu,
	}
}

func (t *ItemTransferTransactor) Accept(ctx context.Context, transferID string) (domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return domain.ItemTransfer{}, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	transfer, err := t.transfers.Get(ctx, transferID)
	if err != nil {
		return domain.ItemTransfer{}, err
	}

	if transfer.Status != domain.TransferStatusPending {
		return domain.ItemTransfer{}, fmt.Errorf("%w: transfer is not pending", domain.ErrValidation)
	}

	item, err := t.items.Get(ctx, transfer.ItemID)
	if err != nil {
		return domain.ItemTransfer{}, err
	}

	if item.OwnerID != transfer.SenderID {
		return domain.ItemTransfer{}, domain.ErrForbidden
	}
	if item.ArchivedAt != nil || item.DisposalReason != nil {
		return domain.ItemTransfer{}, fmt.Errorf("%w: item is not active", domain.ErrValidation)
	}

	if err := t.applyWearLogs(ctx, transfer); err != nil {
		return domain.ItemTransfer{}, err
	}

	if err := t.removeItemFromOutfits(ctx, transfer.SenderID, transfer.ItemID); err != nil {
		return domain.ItemTransfer{}, err
	}

	if err := t.shares.DeleteByTarget(ctx, domain.ShareTargetItem, transfer.ItemID); err != nil {
		return domain.ItemTransfer{}, err
	}

	item.OwnerID = transfer.RecipientID
	item.LocationID = nil
	if err := t.items.Save(ctx, item); err != nil {
		return domain.ItemTransfer{}, err
	}

	now := time.Now()
	transfer.Status = domain.TransferStatusAccepted
	transfer.DecidedAt = &now
	if err := t.transfers.Save(ctx, transfer); err != nil {
		return domain.ItemTransfer{}, err
	}

	return transfer, nil
}

func (t *ItemTransferTransactor) applyWearLogs(ctx context.Context, transfer domain.ItemTransfer) error {
	wls, err := t.wearLogs.ListByItem(ctx, transfer.ItemID)
	if err != nil {
		return err
	}
	if transfer.TransferHistory {
		for i := range wls {
			wls[i].OwnerID = transfer.RecipientID
			if err := t.wearLogs.Save(ctx, wls[i]); err != nil {
				return err
			}
		}
	} else {
		for _, wl := range wls {
			if err := t.wearLogs.Delete(ctx, wl.ID); err != nil {
				return err
			}
			if err := t.outfitLogs.RemoveWearLogLink(ctx, wl.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *ItemTransferTransactor) removeItemFromOutfits(ctx context.Context, ownerID, itemID string) error {
	outfits, err := t.outfits.ListByOwner(ctx, ownerID)
	if err != nil {
		return err
	}
	for _, o := range outfits {
		if outfitContainsItem(o, itemID) {
			if err := t.outfits.DeleteItem(ctx, o.ID, itemID); err != nil {
				return err
			}
		}
	}
	return nil
}

func outfitContainsItem(o domain.Outfit, itemID string) bool {
	for _, oi := range o.Items {
		if oi.ItemID == itemID {
			return true
		}
	}
	return false
}
