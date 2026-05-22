package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// itemTransferTxDB is the subset of *sql.DB used by ItemTransferTransactor.
type itemTransferTxDB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

var _ ports.ItemTransferTransactor = (*ItemTransferTransactor)(nil)

// ItemTransferTransactor is a SQL-backed implementation of ports.ItemTransferTransactor.
type ItemTransferTransactor struct {
	db itemTransferTxDB
}

// NewItemTransferTransactor creates an ItemTransferTransactor backed by the given db.
func NewItemTransferTransactor(db itemTransferTxDB) *ItemTransferTransactor {
	return &ItemTransferTransactor{db: db}
}

// Accept atomically re-reads the transfer and item under a transaction and transfers
// item ownership to the recipient.
func (t *ItemTransferTransactor) Accept(ctx context.Context, transferID string) (domain.ItemTransfer, error) {
	if err := ctx.Err(); err != nil {
		return domain.ItemTransfer{}, err
	}
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ItemTransfer{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback() //nolint:errcheck
		}
	}()

	transfer, err := readAndValidateTransfer(ctx, tx, transferID)
	if err != nil {
		return domain.ItemTransfer{}, err
	}

	if err := validateItemForTransfer(ctx, tx, transfer.ItemID, transfer.SenderID); err != nil {
		return domain.ItemTransfer{}, err
	}

	if err := applyWearLogTransfer(ctx, tx, transfer.ItemID, transfer.RecipientID, transfer.TransferHistory); err != nil {
		return domain.ItemTransfer{}, err
	}

	if err := deleteOutfitItemsForItem(ctx, tx, transfer.ItemID); err != nil {
		return domain.ItemTransfer{}, err
	}

	if err := deleteSharesForItem(ctx, tx, transfer.ItemID); err != nil {
		return domain.ItemTransfer{}, err
	}

	if err := updateItemOwner(ctx, tx, transfer.ItemID, transfer.RecipientID); err != nil {
		return domain.ItemTransfer{}, err
	}

	now := time.Now().UTC()
	if err := acceptTransfer(ctx, tx, transfer.ID, now); err != nil {
		return domain.ItemTransfer{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.ItemTransfer{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	committed = true

	transfer.Status = domain.TransferStatusAccepted
	transfer.DecidedAt = &now
	return transfer, nil
}

// applyWearLogTransfer reassigns or deletes wear logs for the transferred item.
func applyWearLogTransfer(ctx context.Context, tx *sql.Tx, itemID, recipientID string, transferHistory bool) error {
	if transferHistory {
		const q = `UPDATE wear_logs SET owner_id = ? WHERE item_id = ?`
		if _, err := tx.ExecContext(ctx, q, recipientID, itemID); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
	} else {
		const q = `DELETE FROM wear_logs WHERE item_id = ?`
		if _, err := tx.ExecContext(ctx, q, itemID); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
	}
	return nil
}

// deleteOutfitItemsForItem removes all outfit memberships for the item.
func deleteOutfitItemsForItem(ctx context.Context, tx *sql.Tx, itemID string) error {
	const q = `DELETE FROM outfit_items WHERE item_id = ?`
	if _, err := tx.ExecContext(ctx, q, itemID); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// deleteSharesForItem removes all shares targeting the item.
func deleteSharesForItem(ctx context.Context, tx *sql.Tx, itemID string) error {
	const q = `DELETE FROM shares WHERE target_type = 'item' AND target_id = ?`
	if _, err := tx.ExecContext(ctx, q, itemID); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// updateItemOwner sets owner_id to recipientID and clears location_id.
func updateItemOwner(ctx context.Context, tx *sql.Tx, itemID, recipientID string) error {
	const q = `UPDATE items SET owner_id = ?, location_id = NULL WHERE id = ?`
	if _, err := tx.ExecContext(ctx, q, recipientID, itemID); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// acceptTransfer marks the transfer as accepted with decidedAt set.
func acceptTransfer(ctx context.Context, tx *sql.Tx, transferID string, decidedAt time.Time) error {
	const q = `UPDATE item_transfers SET status = 'accepted', decided_at = ? WHERE id = ?`
	if _, err := tx.ExecContext(ctx, q, decidedAt.Format(time.RFC3339Nano), transferID); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// validateItemForTransfer reads the item and asserts ownership and active status.
func validateItemForTransfer(ctx context.Context, tx *sql.Tx, itemID, senderID string) error {
	const q = `SELECT owner_id, archived_at, disposal_reason FROM items WHERE id = ?`
	var (
		ownerID        string
		archivedAt     sql.NullString
		disposalReason sql.NullString
	)
	if err := tx.QueryRowContext(ctx, q, itemID).Scan(&ownerID, &archivedAt, &disposalReason); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: item %s", domain.ErrNotFound, itemID)
		}
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if ownerID != senderID {
		return domain.ErrForbidden
	}
	if archivedAt.Valid || disposalReason.Valid {
		return fmt.Errorf("%w: item is not active", domain.ErrValidation)
	}
	return nil
}

// readAndValidateTransfer reads the transfer row inside the transaction and validates its status.
func readAndValidateTransfer(ctx context.Context, tx *sql.Tx, transferID string) (domain.ItemTransfer, error) {
	const q = `SELECT id, item_id, sender_id, recipient_id, status, transfer_history FROM item_transfers WHERE id = ?`
	var (
		id              string
		itemID          string
		senderID        string
		recipientID     string
		status          string
		transferHistory int
	)
	if err := tx.QueryRowContext(ctx, q, transferID).Scan(&id, &itemID, &senderID, &recipientID, &status, &transferHistory); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ItemTransfer{}, fmt.Errorf("%w: id %s", domain.ErrNotFound, transferID)
		}
		return domain.ItemTransfer{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if domain.TransferStatus(status) != domain.TransferStatusPending {
		return domain.ItemTransfer{}, fmt.Errorf("%w: transfer is not pending", domain.ErrValidation)
	}
	return buildItemTransferFromRow(id, itemID, senderID, recipientID, domain.TransferStatus(status), transferHistory != 0, domain.ItemTransfer{}.CreatedAt, nil), nil
}
