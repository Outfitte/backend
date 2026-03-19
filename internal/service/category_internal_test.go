package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHintShouldMapFieldsWhenGivenKeyLabelPlaceholder(t *testing.T) {
	h := hint("size", "Size", "e.g. M")
	require.Equal(t, "size", h.Key)
	require.Equal(t, "Size", h.Label)
	require.Equal(t, "e.g. M", h.Placeholder)
}
