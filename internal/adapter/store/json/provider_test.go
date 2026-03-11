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

func TestNewProviderShouldImplementStorageProvider(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")
	require.Implements(t, (*ports.StorageProvider[domain.User])(nil), p)
}

func TestListShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := p.List(ctx)
	require.ErrorIs(t, err, context.Canceled)
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

func TestListShouldReturnErrorWhenFileCannotBeOpened(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0o000))
	p := json.NewProvider[domain.User](dir, "users.json")

	_, err := p.List(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
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

func TestListShouldReturnErrorWhenFileContainsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte("not json"), 0o644))
	p := json.NewProvider[domain.User](dir, "users.json")

	_, err := p.List(t.Context())
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

func TestListShouldReturnEmptySliceWhenStoreIsEmpty(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")

	got, err := p.List(t.Context())
	require.NoError(t, err)
	require.Empty(t, got)
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

func TestListShouldReturnAllEntitiesWhenStoreIsPopulated(t *testing.T) {
	dir := t.TempDir()
	p := json.NewProvider[domain.User](dir, "users.json")

	var u1, u2 domain.User
	u1.ID = "1"
	u1.Email = "alice@example.com"
	u2.ID = "2"
	u2.Email = "bob@example.com"
	require.NoError(t, p.Save(t.Context(), u1))
	require.NoError(t, p.Save(t.Context(), u2))

	got, err := p.List(t.Context())
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.ElementsMatch(t, []domain.User{u1, u2}, got)
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

func TestDeleteShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := p.Delete(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnNotFoundWhenStoreIsEmpty(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")

	err := p.Delete(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteShouldReturnNotFoundWhenEntityDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	p := json.NewProvider[domain.User](dir, "users.json")
	var u domain.User
	u.ID = "42"
	require.NoError(t, p.Save(t.Context(), u))

	err := p.Delete(t.Context(), "99")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteShouldReturnErrorWhenFileContainsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte("not json"), 0o644))
	p := json.NewProvider[domain.User](dir, "users.json")

	err := p.Delete(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDeleteShouldReturnErrorWhenFileCannotBeOpened(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0o000))
	p := json.NewProvider[domain.User](dir, "users.json")

	err := p.Delete(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDeleteShouldRemoveEntity(t *testing.T) {
	dir := t.TempDir()
	p := json.NewProvider[domain.User](dir, "users.json")

	var u1, u2 domain.User
	u1.ID = "1"
	u2.ID = "2"
	require.NoError(t, p.Save(t.Context(), u1))
	require.NoError(t, p.Save(t.Context(), u2))

	require.NoError(t, p.Delete(t.Context(), "1"))

	_, err := p.Get(t.Context(), "1")
	require.ErrorIs(t, err, domain.ErrNotFound)

	got, err := p.List(t.Context())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, u2, got[0])
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
