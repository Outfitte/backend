package json

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveFromSliceShouldReturnNilWhenTargetNotFound(t *testing.T) {
	result := removeFromSlice([]string{"a", "b", "c"}, "z")
	require.Nil(t, result)
}

func TestRemoveFromSliceShouldRemoveFirstElement(t *testing.T) {
	result := removeFromSlice([]string{"a", "b", "c"}, "a")
	require.Equal(t, []string{"b", "c"}, result)
}

func TestRemoveFromSliceShouldRemoveMiddleElement(t *testing.T) {
	result := removeFromSlice([]string{"a", "b", "c"}, "b")
	require.Equal(t, []string{"a", "c"}, result)
}

func TestRemoveFromSliceShouldRemoveLastElement(t *testing.T) {
	result := removeFromSlice([]string{"a", "b", "c"}, "c")
	require.Equal(t, []string{"a", "b"}, result)
}

func TestSliceContainsShouldReturnFalseWhenTargetNotFound(t *testing.T) {
	require.False(t, sliceContains([]string{"a", "b"}, "z"))
}

func TestSliceContainsShouldReturnTrueWhenTargetFound(t *testing.T) {
	require.True(t, sliceContains([]string{"a", "b", "c"}, "b"))
}
