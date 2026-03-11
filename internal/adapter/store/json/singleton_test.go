package json_test

import (
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewSingletonStoreShouldImplementSingletonStore(t *testing.T) {
	s := json.NewSingletonStore[domain.AppSettings](t.TempDir(), "app_settings.json")
	require.Implements(t, (*ports.SingletonStore[domain.AppSettings])(nil), s)
}

func TestLoadShouldReturnNotImplemented(t *testing.T) {
	s := json.NewSingletonStore[domain.AppSettings](t.TempDir(), "app_settings.json")

	_, err := s.Load(t.Context())
	require.EqualError(t, err, "not implemented")
}

func TestSingletonSaveShouldReturnNotImplemented(t *testing.T) {
	s := json.NewSingletonStore[domain.AppSettings](t.TempDir(), "app_settings.json")

	err := s.Save(t.Context(), domain.AppSettings{})
	require.EqualError(t, err, "not implemented")
}
