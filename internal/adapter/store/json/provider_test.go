package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewProviderShouldReturnProvider(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")
	require.NotNil(t, p)
}

func TestSaveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := p.Save(ctx, domain.User{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestSaveShouldReturnErrorWhenFileCannotBeOpened(t *testing.T) {
	p := json.NewProvider[domain.User]("/nonexistent/path", "users.json")

	err := p.Save(t.Context(), domain.User{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSaveShouldReturnErrorWhenFileContainsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte("not json"), 0o644))
	p := json.NewProvider[domain.User](dir, "users.json")

	err := p.Save(t.Context(), domain.User{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSaveShouldPersistNewEntity(t *testing.T) {
	dir := t.TempDir()
	p := json.NewProvider[domain.User](dir, "users.json")
	user := domain.User{Email: "alice@example.com"}

	require.NoError(t, p.Save(t.Context(), user))

	data, err := os.ReadFile(filepath.Join(dir, "users.json"))
	require.NoError(t, err)
	require.Contains(t, string(data), "alice@example.com")
}

func TestSaveShouldOverwriteExistingEntity(t *testing.T) {
	dir := t.TempDir()
	p := json.NewProvider[domain.User](dir, "users.json")

	var original domain.User
	original.ID = "42"
	original.Email = "alice@example.com"
	require.NoError(t, p.Save(t.Context(), original))

	var updated domain.User
	updated.ID = "42"
	updated.Email = "alice-updated@example.com"
	require.NoError(t, p.Save(t.Context(), updated))

	data, err := os.ReadFile(filepath.Join(dir, "users.json"))
	require.NoError(t, err)
	require.Contains(t, string(data), "alice-updated@example.com")
	require.NotContains(t, string(data), "alice@example.com")
}
