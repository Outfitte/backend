package local_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/media/local"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

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
