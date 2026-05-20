package ports

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
)

// ItemTransferTransactor is the port for atomic item transfer operations.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type ItemTransferTransactor interface {
	// Accept atomically re-reads the transfer and item under a transaction,
	// removes the item's wear history (when TransferHistory is false), removes
	// the item from all outfit memberships, removes all shares for the item,
	// updates the item owner to the recipient, and marks the transfer as
	// accepted with DecidedAt set. Returns the updated ItemTransfer with
	// Status=accepted and DecidedAt populated.
	//
	// All preconditions are re-checked inside the transaction:
	//   - transfer must exist            -> domain.ErrNotFound if missing
	//   - transfer must be pending       -> domain.ErrValidation if status != pending
	//   - item must be active            -> domain.ErrNotFound if missing or inactive
	//   - sender must still own the item -> domain.ErrForbidden
	//
	// Only the transfer ID is accepted; all other state is read fresh inside
	// the transaction to avoid TOCTOU races.
	Accept(ctx context.Context, transferID string) (domain.ItemTransfer, error)
}
