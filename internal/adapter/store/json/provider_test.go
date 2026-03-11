package json_test

import (
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewProviderShouldReturnProvider(t *testing.T) {
	p := json.NewProvider[domain.User](t.TempDir(), "users.json")
	require.NotNil(t, p)
}
