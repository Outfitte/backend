package main

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return strconv.Itoa(port)
}

func TestRunShouldShutdownCleanlyWhenContextCancelled(t *testing.T) {
	port := freePort(t)
	t.Setenv("STORAGE_DATA_PATH", t.TempDir())
	t.Setenv("MEDIA_STORAGE_PATH", t.TempDir())
	t.Setenv("SERVER_PORT", port)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- run(ctx)
	}()

	addr := "http://localhost:" + port + "/health"
	var resp *http.Response
	for range 20 {
		var err error
		resp, err = http.Get(addr)
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
