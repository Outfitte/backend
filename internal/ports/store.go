package ports

import "context"

// Entity is the constraint for types that can be stored via StorageProvider.
// The GetID method must return the string UUID that uniquely identifies the entity.
type Entity interface {
	GetID() string
}

// StorageProvider is a generic CRUD interface for persisting entities of type T.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
//
// Save is an upsert keyed on entity.GetID(): it creates the record if no record
// with that id exists, or replaces it if one does.
type StorageProvider[T Entity] interface {
	// Get retrieves the entity with the given id.
	// Returns a domain not-found error if no entity with that id exists.
	Get(ctx context.Context, id string) (T, error)

	// List returns all stored entities in an unspecified order.
	List(ctx context.Context) ([]T, error)

	// Save creates or replaces the entity identified by entity.GetID().
	Save(ctx context.Context, entity T) error

	// Delete removes the entity with the given id.
	// Returns a domain not-found error if no entity with that id exists.
	Delete(ctx context.Context, id string) error
}
