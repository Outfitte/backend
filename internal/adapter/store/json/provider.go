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

func upsert[T ports.Entity](entities []T, entity T) []T {
	for i, e := range entities {
		if e.GetID() == entity.GetID() {
			entities[i] = entity
			return entities
		}
	}
	return append(entities, entity)
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

	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if err := json.NewEncoder(f).Encode(entities); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return nil
}
