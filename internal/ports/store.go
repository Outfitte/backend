package ports

import "context"

// StorageProvider is a generic CRUD interface for persisting entities of type T.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
//
// The id argument in Get, Save, and Delete is a string UUID identifying the
// entity. Save is an upsert: it creates the record if no record with the given
// id exists, or replaces it if one does.
type StorageProvider[T any] interface {
	// Get retrieves the entity with the given id.
	// Returns a domain not-found error if no entity with that id exists.
	Get(ctx context.Context, id string) (T, error)

	// List returns all stored entities in an unspecified order.
	List(ctx context.Context) ([]T, error)

	// Save creates or replaces the entity with the given id.
	Save(ctx context.Context, id string, entity T) error

	// Delete removes the entity with the given id.
	// Returns a domain not-found error if no entity with that id exists.
	Delete(ctx context.Context, id string) error
}
