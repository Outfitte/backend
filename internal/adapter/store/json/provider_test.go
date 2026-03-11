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

func TestSaveShouldReturnErrorWhenFileCannotBeOpened(t *testing.T) {
	p := json.NewProvider[domain.User]("/nonexistent/path", "users.json")

	err := p.Save(context.Background(), domain.User{})
	require.ErrorContains(t, err, "no such file or directory")
}

func TestSaveShouldReturnErrorWhenFileContainsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte("not json"), 0o644))
	p := json.NewProvider[domain.User](dir, "users.json")

	err := p.Save(context.Background(), domain.User{})
	require.ErrorContains(t, err, "invalid character")
}

func TestSaveShouldPersistNewEntity(t *testing.T) {
	dir := t.TempDir()
	p := json.NewProvider[domain.User](dir, "users.json")
	user := domain.User{Email: "alice@example.com"}

	require.NoError(t, p.Save(context.Background(), user))

	data, err := os.ReadFile(filepath.Join(dir, "users.json"))
	require.NoError(t, err)
	require.Contains(t, string(data), "alice@example.com")
}

func TestSaveShouldOverwriteExistingEntity(t *testing.T) {
	dir := t.TempDir()
	p := json.NewProvider[domain.User](dir, "users.json")
	original := domain.User{Email: "alice@example.com"}
	require.NoError(t, p.Save(context.Background(), original))

	updated := domain.User{Email: "alice-updated@example.com"}
	require.NoError(t, p.Save(context.Background(), updated))

	data, err := os.ReadFile(filepath.Join(dir, "users.json"))
	require.NoError(t, err)
	require.Contains(t, string(data), "alice-updated@example.com")
	require.NotContains(t, string(data), "alice@example.com")
}
