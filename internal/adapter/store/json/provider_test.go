package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	storejson "github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
)

type testEntity struct {
	ID    string
	Value string
}

func (e testEntity) GetID() string { return e.ID }

func newProvider(t *testing.T) *storejson.Provider[testEntity] {
	t.Helper()
	return storejson.NewProvider[testEntity](t.TempDir(), "entities.json")
}

func TestProvider_Get_NotFound(t *testing.T) {
	p := newProvider(t)
	_, err := p.Get(context.Background(), "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestProvider_Save_AndGet(t *testing.T) {
	p := newProvider(t)
	e := testEntity{ID: "abc", Value: "hello"}
	require.NoError(t, p.Save(context.Background(), e))
	got, err := p.Get(context.Background(), "abc")
	require.NoError(t, err)
	require.Equal(t, e, got)
}

func TestProvider_List_Empty(t *testing.T) {
	p := newProvider(t)
	list, err := p.List(context.Background())
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestProvider_List(t *testing.T) {
	p := newProvider(t)
	entities := []testEntity{
		{ID: "1", Value: "a"},
		{ID: "2", Value: "b"},
		{ID: "3", Value: "c"},
	}
	for _, e := range entities {
		require.NoError(t, p.Save(context.Background(), e))
	}
	list, err := p.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 3)
}

func TestProvider_Save_Upsert(t *testing.T) {
	p := newProvider(t)
	e1 := testEntity{ID: "abc", Value: "first"}
	e2 := testEntity{ID: "abc", Value: "second"}
	require.NoError(t, p.Save(context.Background(), e1))
	require.NoError(t, p.Save(context.Background(), e2))
	got, err := p.Get(context.Background(), "abc")
	require.NoError(t, err)
	require.Equal(t, e2, got)
}

func TestProvider_Delete_Panics(t *testing.T) {
	p := newProvider(t)
	require.Panics(t, func() {
		_ = p.Delete(context.Background(), "any")
	})
}

// newProviderWithDirAsFile returns a provider whose file path is occupied by a
// directory, causing os.ReadFile to fail with a non-ErrNotExist error.
func newProviderWithDirAsFile(t *testing.T) *storejson.Provider[testEntity] {
	t.Helper()
	tmp := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmp, "entities.json"), 0o755))
	return storejson.NewProvider[testEntity](tmp, "entities.json")
}

func TestProvider_Get_LoadReadError(t *testing.T) {
	p := newProviderWithDirAsFile(t)
	_, err := p.Get(context.Background(), "any")
	require.ErrorIs(t, err, domain.ErrInternal)
}

func TestProvider_Get_LoadUnmarshalError(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "entities.json"), []byte("not-json"), 0o600))
	p := storejson.NewProvider[testEntity](tmp, "entities.json")
	_, err := p.Get(context.Background(), "any")
	require.ErrorIs(t, err, domain.ErrInternal)
}

func TestProvider_Save_LoadReadError(t *testing.T) {
	p := newProviderWithDirAsFile(t)
	err := p.Save(context.Background(), testEntity{ID: "1", Value: "v"})
	require.ErrorIs(t, err, domain.ErrInternal)
}

func TestProvider_List_LoadReadError(t *testing.T) {
	p := newProviderWithDirAsFile(t)
	_, err := p.List(context.Background())
	require.ErrorIs(t, err, domain.ErrInternal)
}

func TestProvider_Save_PersistMkdirError(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "sub")
	p := storejson.NewProvider[testEntity](root, "entities.json")
	// chmod 0o111 (execute-only): traversal works so load gets ENOENT and
	// returns an empty map, but MkdirAll cannot create new entries in tmp.
	require.NoError(t, os.Chmod(tmp, 0o111))
	t.Cleanup(func() { _ = os.Chmod(tmp, 0o755) })
	err := p.Save(context.Background(), testEntity{ID: "1", Value: "v"})
	require.ErrorIs(t, err, domain.ErrInternal)
}

func TestProvider_Save_PersistWriteError(t *testing.T) {
	tmp := t.TempDir()
	p := storejson.NewProvider[testEntity](tmp, "entities.json")
	// First save creates the file.
	require.NoError(t, p.Save(context.Background(), testEntity{ID: "1", Value: "v"}))
	// Make file read-only: load can still read it, but WriteFile will fail.
	filePath := filepath.Join(tmp, "entities.json")
	require.NoError(t, os.Chmod(filePath, 0o400))
	t.Cleanup(func() { _ = os.Chmod(filePath, 0o644) })
	err := p.Save(context.Background(), testEntity{ID: "2", Value: "w"})
	require.ErrorIs(t, err, domain.ErrInternal)
}

// unmarshalableEntity implements ports.Entity but cannot be marshalled to JSON
// because it contains a channel field.
type unmarshalableEntity struct {
	ID  string
	Bad chan int
}

func (e unmarshalableEntity) GetID() string { return e.ID }

func TestProvider_Save_MarshalError(t *testing.T) {
	p := storejson.NewProvider[unmarshalableEntity](t.TempDir(), "entities.json")
	err := p.Save(context.Background(), unmarshalableEntity{ID: "1", Bad: make(chan int)})
	require.ErrorIs(t, err, domain.ErrInternal)
}
