package json

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
)

// SingletonStore is a JSON file-backed implementation of ports.SingletonStore[T].
// The file contains a single JSON object (no map wrapper).
type SingletonStore[T any] struct {
	path string
	mu   sync.Mutex
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
func (s *SingletonStore[T]) Load(_ context.Context) (T, error) {
	var zero T
	return zero, errors.New("not implemented")
}

// Save persists the singleton value, replacing any previously saved value.
func (s *SingletonStore[T]) Save(_ context.Context, _ T) error {
	return errors.New("not implemented")
}

