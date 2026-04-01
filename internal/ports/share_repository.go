package ports

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
)

// ShareRepository is the storage port for Share persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type ShareRepository interface {
	// Get retrieves a single share by ID.
	// Returns domain.ErrNotFound if no share with that ID exists.
	Get(ctx context.Context, id string) (domain.Share, error)

	// Save creates or updates a share entry.
	Save(ctx context.Context, share domain.Share) error

	// Delete removes the share identified by id.
	// Returns domain.ErrNotFound if no share with that ID exists.
	Delete(ctx context.Context, id string) error

	// ListByOwner returns all outgoing shares created by ownerID.
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Share, error)

	// ListByRecipient returns all incoming shares for recipientID.
	ListByRecipient(ctx context.Context, recipientID string) ([]domain.Share, error)

	// FindByTarget returns the share matching the given owner, recipient, target type,
	// and target ID, or nil if no such entry exists. Used for duplicate detection.
	// Returns nil (not domain.ErrNotFound) when no match is found.
	FindByTarget(ctx context.Context, ownerID, recipientID string, targetType domain.ShareTargetType, targetID string) (*domain.Share, error)

	// DeleteByTarget removes all shares for the given target type and target ID.
	// No error is returned if no entries exist. Called when the target entity is deleted.
	DeleteByTarget(ctx context.Context, targetType domain.ShareTargetType, targetID string) error

	// HasDirectAccess reports whether a share entry exists granting recipientID access
	// to the exact target. It does not walk location ancestors — that logic lives in the
	// service layer.
	HasDirectAccess(ctx context.Context, recipientID string, targetType domain.ShareTargetType, targetID string) (bool, error)

	// ListByRecipientAndType returns all incoming shares of a specific type for recipientID.
	// Used by the hydrated shared-with-me endpoint.
	ListByRecipientAndType(ctx context.Context, recipientID string, targetType domain.ShareTargetType) ([]domain.Share, error)
}
