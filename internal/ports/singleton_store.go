package ports

import "context"

// SingletonStore is a storage interface for singleton values — values where there
// is always exactly one instance (e.g. AppSettings).
//
// Because singleton values have no identity key, they do not satisfy the Entity
// constraint and cannot be stored via StorageProvider.
//
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type SingletonStore[T any] interface {
	// Load retrieves the singleton value.
	// Returns a domain not-found error if no value has been saved yet.
	Load(ctx context.Context) (T, error)

	// Save persists the singleton value, replacing any previously saved value.
	Save(ctx context.Context, value T) error
}
