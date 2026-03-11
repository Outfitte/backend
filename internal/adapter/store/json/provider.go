package json

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"

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

// Save creates or replaces the entity identified by entity.GetID().
func (p *Provider[T]) Save(_ context.Context, entity T) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	f, err := os.OpenFile(p.path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var entities []T
	if len(data) > 0 {
		if err := json.Unmarshal(data, &entities); err != nil {
			return err
		}
	}

	replaced := false
	for i, e := range entities {
		if e.GetID() == entity.GetID() {
			entities[i] = entity
			replaced = true
			break
		}
	}
	if !replaced {
		entities = append(entities, entity)
	}

	out, err := json.Marshal(entities)
	if err != nil {
		return err
	}

	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.WriteAt(out, 0); err != nil {
		return err
	}

	return nil
}
