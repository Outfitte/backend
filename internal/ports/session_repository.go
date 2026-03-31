package ports

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
)

// SessionRepository is the storage port for Session persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type SessionRepository interface {
	// Get retrieves a single session by ID.
	// Returns domain.ErrNotFound if no session with that ID exists.
	Get(ctx context.Context, id string) (domain.Session, error)

	// Save creates or updates the session.
	Save(ctx context.Context, session domain.Session) error

	// Delete removes the session identified by id.
	// Returns domain.ErrNotFound if no session with that ID exists.
	Delete(ctx context.Context, id string) error

	// FindByTokenHash retrieves a session by its token hash.
	// Returns domain.ErrNotFound if no session with that hash exists.
	FindByTokenHash(ctx context.Context, hash string) (domain.Session, error)

	// CountByUser returns the total number of sessions for the given userID.
	// Returns 0, nil if the user has no sessions or does not exist.
	CountByUser(ctx context.Context, userID string) (int, error)

	// DeleteOldestByUser removes the oldest session belonging to userID.
	// Returns domain.ErrNotFound if the user has no sessions.
	DeleteOldestByUser(ctx context.Context, userID string) error
}
