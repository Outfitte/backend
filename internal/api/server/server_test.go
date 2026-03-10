package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/outfitte/outfitte/internal/api/server"
	"github.com/outfitte/outfitte/internal/config"
)

func TestNewShouldServeHealthWhenGivenValidConfig(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}

	srv, err := server.New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
