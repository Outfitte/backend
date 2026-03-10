package main

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
	l.Close()
	return port
}

func TestRunShouldReturnErrorWhenConfigInvalid(t *testing.T) {
	err := run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration error")
}

func TestRunShouldShutdownCleanlyWhenContextCancelled(t *testing.T) {
	port := freePort(t)
	t.Setenv("STORAGE_DATA_PATH", t.TempDir())
	t.Setenv("MEDIA_STORAGE_PATH", t.TempDir())
	t.Setenv("SERVER_PORT", port)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- run(ctx) }()

	addr := "http://localhost:" + port + "/health"
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
