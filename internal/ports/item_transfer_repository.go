package ports

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
)

// ItemTransferRepository is the storage port for ItemTransfer persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type ItemTransferRepository interface {
	// Get retrieves a single transfer by ID.
	// Returns domain.ErrNotFound if no transfer with that ID exists.
	Get(ctx context.Context, id string) (domain.ItemTransfer, error)

	// Save creates or updates a transfer entry.
	Save(ctx context.Context, transfer domain.ItemTransfer) error

	// Delete removes the transfer identified by id.
	// Returns domain.ErrNotFound if no transfer with that ID exists.
	Delete(ctx context.Context, id string) error

	// ListBySender returns outgoing transfers created by senderID.
	// A nil statusFilter returns all transfers regardless of status.
	ListBySender(ctx context.Context, senderID string, statusFilter *domain.TransferStatus) ([]domain.ItemTransfer, error)

	// ListByRecipient returns incoming transfers for recipientID.
	// A nil statusFilter returns all transfers regardless of status.
	ListByRecipient(ctx context.Context, recipientID string, statusFilter *domain.TransferStatus) ([]domain.ItemTransfer, error)

	// FindPendingByItem returns the pending transfer for itemID, or nil if none exists.
	// Returns nil (not domain.ErrNotFound) when no pending transfer is found.
	FindPendingByItem(ctx context.Context, itemID string) (*domain.ItemTransfer, error)

	// HasPending reports whether a pending transfer exists for itemID.
	HasPending(ctx context.Context, itemID string) (bool, error)
}
