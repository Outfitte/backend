package json

import (
	"context"
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

func TestSingletonSaveShouldReturnNotImplemented(t *testing.T) {
	s := NewSingletonStore[domain.AppSettings](t.TempDir(), "app_settings.json")

	err := s.Save(t.Context(), domain.AppSettings{})
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
