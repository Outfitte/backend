package local_test

import (
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/media/local"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewProviderShouldImplementMediaProvider(t *testing.T) {
	p := local.NewProvider(t.TempDir())
	require.Implements(t, (*ports.MediaProvider)(nil), p)
}
