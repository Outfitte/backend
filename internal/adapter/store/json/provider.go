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

	f, err := os.Open(p.path)
	if err != nil {
		return zero, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer f.Close()

	var entities []T
	if err := json.NewDecoder(f).Decode(&entities); err != nil && !errors.Is(err, io.EOF) {
		return zero, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	for _, e := range entities {
		if e.GetID() == id {
			return e, nil
		}
	}

	return zero, fmt.Errorf("%w: id %s", domain.ErrNotFound, id)
}

// Save creates or replaces the entity identified by entity.GetID().
func (p *Provider[T]) Save(ctx context.Context, entity T) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	f, err := os.OpenFile(p.path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer f.Close()

	var entities []T
	if err := json.NewDecoder(f).Decode(&entities); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	entities = upsert(entities, entity)

	if err := writeJSON(f, entities); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return nil
}
