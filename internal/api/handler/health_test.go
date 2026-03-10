package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/outfitte/outfitte/internal/api/handler"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
