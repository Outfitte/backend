package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewAppSettingsRepositoryShouldImplementAppSettingsRepository(t *testing.T) {
	r := json.NewAppSettingsRepository(t.TempDir())
	require.Implements(t, (*ports.AppSettingsRepository)(nil), r)
}

func TestLoadShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app_settings.json"), []byte("not json"), 0o644))
	r := json.NewAppSettingsRepository(dir)

	_, err := r.Load(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLoadShouldReturnNotFoundWhenNoSettingsSaved(t *testing.T) {
	r := json.NewAppSettingsRepository(t.TempDir())

	_, err := r.Load(t.Context())
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSaveShouldReturnErrorWhenContextIsCancelledForAppSettings(t *testing.T) {
	r := json.NewAppSettingsRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.AppSettings{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestLoadShouldReturnSettingsWhenSaved(t *testing.T) {
	r := json.NewAppSettingsRepository(t.TempDir())
	settings := domain.AppSettings{RegistrationEnabled: true}
	require.NoError(t, r.Save(t.Context(), settings))

	got, err := r.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, settings, got)
}

func TestLoadShouldReturnErrorWhenContextIsCancelledForAppSettings(t *testing.T) {
	r := json.NewAppSettingsRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Load(ctx)
	require.ErrorIs(t, err, context.Canceled)
}
