package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/outfitte/outfitte/internal/config"
	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func TestRunShouldShutdownCleanlyWhenContextCancelled(t *testing.T) {
	port := freePort(t)
	cfg := &config.Config{
		StorageDataPath:  t.TempDir(),
		MediaStoragePath: t.TempDir(),
		ServerPort:       strconv.Itoa(port),
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- runServer(ctx, cfg, slog.New(slog.DiscardHandler)) }()

	addr := "http://localhost:" + strconv.Itoa(port) + "/health"
	require.Eventually(t, func() bool {
		resp, err := http.Get(addr)
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 10*time.Millisecond)

	cancel()
	require.NoError(t, <-done)
}
