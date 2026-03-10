package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/stretchr/testify/require"
)

func TestHealthShouldReturn200WhenCalled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
