package json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewSingletonStoreShouldImplementSingletonStore(t *testing.T) {
	s := NewSingletonStore[domain.AppSettings](t.TempDir(), "app_settings.json")
	require.Implements(t, (*ports.SingletonStore[domain.AppSettings])(nil), s)
}

func TestLoadShouldReturnNotImplemented(t *testing.T) {
	s := NewSingletonStore[domain.AppSettings](t.TempDir(), "app_settings.json")

	_, err := s.Load(t.Context())
	require.ErrorIs(t, err, errNotImplemented)
}

func TestLoadShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	s := NewSingletonStore[domain.AppSettings](t.TempDir(), "app_settings.json")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := s.Load(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSingletonSaveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	s := NewSingletonStore[domain.AppSettings](t.TempDir(), "app_settings.json")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := s.Save(ctx, domain.AppSettings{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestSingletonSaveShouldReturnIOErrorWhenFileCannotBeOpened(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o000))
	s := NewSingletonStore[domain.AppSettings](dir, "app_settings.json")

	err := s.Save(t.Context(), domain.AppSettings{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSingletonSaveShouldPersistValueWhenSuccessful(t *testing.T) {
	dir := t.TempDir()
	s := NewSingletonStore[domain.AppSettings](dir, "app_settings.json")
	settings := domain.AppSettings{RegistrationEnabled: true}

	require.NoError(t, s.Save(t.Context(), settings))

	data, err := os.ReadFile(filepath.Join(dir, "app_settings.json"))
	require.NoError(t, err)
	var got domain.AppSettings
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, settings, got)
}

func TestSingletonSaveShouldBeThreadSafeWhenCalledConcurrently(t *testing.T) {
	s := NewSingletonStore[domain.AppSettings](t.TempDir(), "app_settings.json")
	const goroutines = 20

	errs := make(chan error, goroutines)
	for range goroutines {
		go func() {
			errs <- s.Save(t.Context(), domain.AppSettings{RegistrationEnabled: true})
		}()
	}

	for range goroutines {
		require.NoError(t, <-errs)
	}
}
