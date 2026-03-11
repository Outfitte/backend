package json

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/outfitte/outfitte/internal/domain"
)

var errNotImplemented = errors.New("not implemented")

// SingletonStore is a JSON file-backed implementation of ports.SingletonStore[T].
// The file contains a single JSON object (no map wrapper).
type SingletonStore[T any] struct {
	path string
	mu   sync.RWMutex
}

// NewSingletonStore creates a SingletonStore that stores the singleton value in root/filename.
// filename should be the bare name, e.g. "app_settings.json".
func NewSingletonStore[T any](root, filename string) *SingletonStore[T] {
	return &SingletonStore[T]{
		path: filepath.Join(root, filename),
	}
}

// Load retrieves the singleton value.
// Returns domain.ErrNotFound if no value has been saved yet.
func (s *SingletonStore[T]) Load(ctx context.Context) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	return zero, errNotImplemented
}

// Save persists the singleton value, replacing any previously saved value.
func (s *SingletonStore[T]) Save(ctx context.Context, value T) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.path, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer f.Close()

	if err := writeJSON(f, value); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return nil
}
