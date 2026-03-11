package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// Provider is a JSON file-backed implementation of ports.StorageProvider[T].
// All entities of type T are stored in a single JSON file at root/filename.
type Provider[T ports.Entity] struct {
	path string
	mu   sync.RWMutex
}

// NewProvider creates a Provider that stores entities in root/filename.
// filename should be the bare name, e.g. "users.json".
func NewProvider[T ports.Entity](root, filename string) *Provider[T] {
	return &Provider[T]{
		path: filepath.Join(root, filename),
	}
}

// openFile opens the store file with the given flags.
// Returns domain.ErrNotFound if the file does not exist, or domain.ErrIO for any other OS error.
func (p *Provider[T]) openFile(flag int) (*os.File, error) {
	f, err := os.OpenFile(p.path, flag, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return f, nil
}

// decodeEntities reads and decodes JSON entities from f.
// Returns domain.ErrIO on decode failure.
func decodeEntities[T any](f *os.File) ([]T, error) {
	entities := []T{}
	if err := json.NewDecoder(f).Decode(&entities); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return entities, nil
}

func writeJSON(f *os.File, v any) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(v)
}

func upsert[T ports.Entity](entities []T, entity T) []T {
	for i, e := range entities {
		if e.GetID() == entity.GetID() {
			entities[i] = entity
			return entities
		}
	}
	return append(entities, entity)
}

// Get retrieves the entity with the given id.
// Returns domain.ErrNotFound if no entity with that id exists.
func (p *Provider[T]) Get(ctx context.Context, id string) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	f, err := p.openFile(os.O_RDONLY)
	if err != nil {
		return zero, err
	}
	defer f.Close()

	entities, err := decodeEntities[T](f)
	if err != nil {
		return zero, err
	}

	for _, e := range entities {
		if e.GetID() == id {
			return e, nil
		}
	}

	return zero, fmt.Errorf("%w: id %s", domain.ErrNotFound, id)
}

// List returns all stored entities in an unspecified order.
// Returns an empty slice if the store file does not exist yet.
func (p *Provider[T]) List(ctx context.Context) ([]T, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	f, err := p.openFile(os.O_RDONLY)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return []T{}, nil
		}
		return nil, err
	}
	defer f.Close()

	return decodeEntities[T](f)
}

// Delete removes the entity with the given id.
// Returns domain.ErrNotFound if no entity with that id exists.
func (p *Provider[T]) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	f, err := p.openFile(os.O_RDWR)
	if err != nil {
		return err
	}
	defer f.Close()

	entities, err := decodeEntities[T](f)
	if err != nil {
		return err
	}

	found := false
	filtered := make([]T, 0, len(entities))
	for _, e := range entities {
		if e.GetID() == id {
			found = true
		} else {
			filtered = append(filtered, e)
		}
	}
	if !found {
		return fmt.Errorf("%w: id %s", domain.ErrNotFound, id)
	}

	if err := writeJSON(f, filtered); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return nil
}

// Save creates or replaces the entity identified by entity.GetID().
func (p *Provider[T]) Save(ctx context.Context, entity T) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Save uses O_CREATE, so ErrNotExist means a broken environment (wrong dir),
	// not a missing entity — all open errors are ErrIO here.
	f, err := os.OpenFile(p.path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer f.Close()

	entities, err := decodeEntities[T](f)
	if err != nil {
		return err
	}

	entities = upsert(entities, entity)

	if err := writeJSON(f, entities); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return nil
}
