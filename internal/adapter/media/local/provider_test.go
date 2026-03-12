package local_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/media/local"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

// errReader is a reader that always returns an error.
type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read failure")
}

func TestNewProviderShouldImplementMediaProvider(t *testing.T) {
	p := local.NewProvider(t.TempDir())
	require.Implements(t, (*ports.MediaProvider)(nil), p)
}

func TestUploadShouldReturnErrWhenContextIsCancelled(t *testing.T) {
	p := local.NewProvider(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := p.Upload(ctx, "image.jpg", strings.NewReader("data"))
	require.ErrorIs(t, err, context.Canceled)
}

func TestUploadShouldReturnErrIOWhenMkdirAllFails(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0o555))
	t.Cleanup(func() { os.Chmod(root, 0o755) })

	p := local.NewProvider(root)
	err := p.Upload(t.Context(), "nested/dir/image.jpg", strings.NewReader("data"))
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUploadShouldReturnErrIOWhenCreateFails(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0o555))
	t.Cleanup(func() { os.Chmod(root, 0o755) })

	p := local.NewProvider(root)
	// flat key — MkdirAll(root) succeeds (dir exists), os.Create fails (read-only)
	err := p.Upload(t.Context(), "image.jpg", strings.NewReader("data"))
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUploadShouldReturnErrIOWhenCopyFails(t *testing.T) {
	p := local.NewProvider(t.TempDir())
	err := p.Upload(t.Context(), "image.jpg", errReader{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUploadShouldWriteFileWithCorrectContentsWhenSuccessful(t *testing.T) {
	root := t.TempDir()
	p := local.NewProvider(root)

	content := "hello upload"
	err := p.Upload(t.Context(), "sub/image.jpg", strings.NewReader(content))
	require.NoError(t, err)

	got, err := os.ReadFile(filepath.Join(root, "sub/image.jpg"))
	require.NoError(t, err)
	require.Equal(t, content, string(got))
}

func TestUploadShouldWriteEmptyFileWhenReaderIsEmpty(t *testing.T) {
	root := t.TempDir()
	p := local.NewProvider(root)

	err := p.Upload(t.Context(), "empty.bin", strings.NewReader(""))
	require.NoError(t, err)

	got, err := os.ReadFile(filepath.Join(root, "empty.bin"))
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestGetURLShouldReturnErrWhenContextIsCancelled(t *testing.T) {
	p := local.NewProvider(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := p.GetURL(ctx, "image.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetURLShouldReturnMediaPathWhenSuccessful(t *testing.T) {
	p := local.NewProvider(t.TempDir())

	url, err := p.GetURL(t.Context(), "sub/image.jpg")
	require.NoError(t, err)
	require.Equal(t, "/media/sub/image.jpg", url)
}

func TestUploadShouldWriteLargeFileWhenContentExceedsBuffer(t *testing.T) {
	root := t.TempDir()
	p := local.NewProvider(root)

	// 2 MB of data — well beyond any single io.Copy buffer
	content := bytes.Repeat([]byte("x"), 2*1024*1024)
	err := p.Upload(t.Context(), "large.bin", bytes.NewReader(content))
	require.NoError(t, err)

	got, err := os.ReadFile(filepath.Join(root, "large.bin"))
	require.NoError(t, err)
	require.Equal(t, content, got)
}
