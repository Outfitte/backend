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

func TestGetShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := p.Get(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSaveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := p.Save(ctx, domain.User{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetShouldReturnNotFoundWhenStoreIsEmpty(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")

	_, err := p.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetShouldReturnErrorWhenFileCannotBeOpened(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0o000))
	p := json.NewProvider[domain.User](dir, "users.json")

	_, err := p.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSaveShouldReturnErrorWhenFileCannotBeOpened(t *testing.T) {
	p := json.NewProvider[domain.User]("/nonexistent/path", "users.json")

	err := p.Save(t.Context(), domain.User{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetShouldReturnErrorWhenFileContainsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte("not json"), 0o644))
	p := json.NewProvider[domain.User](dir, "users.json")

	_, err := p.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetShouldReturnNotFoundWhenEntityDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	p := json.NewProvider[domain.User](dir, "users.json")
	var u domain.User
	u.ID = "42"
	require.NoError(t, p.Save(t.Context(), u))

	_, err := p.Get(t.Context(), "99")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSaveShouldReturnErrorWhenFileContainsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte("not json"), 0o644))
	p := json.NewProvider[domain.User](dir, "users.json")

	err := p.Save(t.Context(), domain.User{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetShouldReturnEntityWhenFound(t *testing.T) {
	dir := t.TempDir()
	p := json.NewProvider[domain.User](dir, "users.json")
	var u domain.User
	u.ID = "42"
	u.Email = "alice@example.com"
	require.NoError(t, p.Save(t.Context(), u))

	got, err := p.Get(t.Context(), "42")
	require.NoError(t, err)
	require.Equal(t, u, got)
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
