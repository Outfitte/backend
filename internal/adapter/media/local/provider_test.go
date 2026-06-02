package local_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/adapter/media/local"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
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

func TestDownloadShouldReturnErrWhenContextIsCancelled(t *testing.T) {
	p := local.NewProvider(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := p.Download(ctx, "image.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDownloadShouldReturnErrIOWhenOpenFails(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "locked.jpg")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0o000))
	t.Cleanup(func() { os.Chmod(path, 0o644) })

	p := local.NewProvider(root)
	_, err := p.Download(t.Context(), "locked.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDownloadShouldReturnErrNotFoundWhenFileDoesNotExist(t *testing.T) {
	p := local.NewProvider(t.TempDir())

	_, err := p.Download(t.Context(), "missing.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDownloadShouldReturnReadableContentWhenSuccessful(t *testing.T) {
	root := t.TempDir()
	p := local.NewProvider(root)

	content := "hello download"
	require.NoError(t, os.WriteFile(filepath.Join(root, "image.jpg"), []byte(content), 0o644))

	rc, err := p.Download(t.Context(), "image.jpg")
	require.NoError(t, err)
	t.Cleanup(func() { rc.Close() })

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, content, string(got))
}

func TestDeleteShouldReturnErrWhenContextIsCancelled(t *testing.T) {
	p := local.NewProvider(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := p.Delete(ctx, "image.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnErrIOWhenRemoveFails(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "image.jpg"), []byte("data"), 0o644))
	require.NoError(t, os.Chmod(root, 0o555))
	t.Cleanup(func() { os.Chmod(root, 0o755) })

	p := local.NewProvider(root)
	err := p.Delete(t.Context(), "image.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDeleteShouldReturnErrNotFoundWhenFileDoesNotExist(t *testing.T) {
	p := local.NewProvider(t.TempDir())

	err := p.Delete(t.Context(), "missing.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteShouldRemoveFileWhenSuccessful(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "image.jpg")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0o644))

	p := local.NewProvider(root)
	require.NoError(t, p.Delete(t.Context(), "image.jpg"))

	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestProviderShouldCompleteFullCycleWhenOperationsAreSequenced(t *testing.T) {
	root := t.TempDir()
	p := local.NewProvider(root)
	ctx := t.Context()

	const key = "cycle/image.jpg"
	content := "full cycle content"

	// Upload
	err := p.Upload(ctx, key, strings.NewReader(content))
	require.NoError(t, err)

	// GetURL
	url, err := p.GetURL(ctx, key)
	require.NoError(t, err)
	require.Equal(t, "/media/"+key, url)

	// Download and verify content
	rc, err := p.Download(ctx, key)
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, rc.Close())
	require.NoError(t, err)
	require.Equal(t, content, string(got))

	// Delete
	require.NoError(t, p.Delete(ctx, key))

	// Verify gone
	_, err = p.Download(ctx, key)
	require.ErrorIs(t, err, domain.ErrNotFound)
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
