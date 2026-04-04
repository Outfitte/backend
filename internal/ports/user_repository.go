package ports

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
)

// UserRepository is the storage port for User persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type UserRepository interface {
	// Get retrieves a single user by ID.
	// Returns domain.ErrNotFound if no user with that ID exists.
	Get(ctx context.Context, id string) (domain.User, error)

	// Save creates or updates the user.
	// Returns domain.ErrConflict if the email is already in use by a different user.
	Save(ctx context.Context, user domain.User) error

	// GetByEmail retrieves a user by email address.
	// Returns domain.ErrNotFound if no user with that email exists.
	GetByEmail(ctx context.Context, email string) (domain.User, error)

	// Count returns the total number of users.
	Count(ctx context.Context) (int, error)

	// List returns all users.
	List(ctx context.Context) ([]domain.User, error)
}
