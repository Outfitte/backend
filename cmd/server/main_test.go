package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	t.Setenv("STORAGE_DATA_PATH", t.TempDir())
	t.Setenv("MEDIA_STORAGE_PATH", t.TempDir())
	t.Setenv("SERVER_PORT", "18080")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- run(ctx)
	}()

	// Wait for server to be ready
	var resp *http.Response
	for range 10 {
		var err error
		resp, err = http.Get("http://localhost:18080/health")
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if resp == nil {
		t.Fatal("server did not start")
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}
