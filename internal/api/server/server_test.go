package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/outfitte/outfitte/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShouldServeHealthWhenGivenValidConfig(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	srv, err := New(cfg)
	require.NoError(t, err)
	assert.Equal(t, ":8080", srv.Addr)

	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// idleListener blocks Accept on a channel; Close unblocks it.
type idleListener struct {
	once sync.Once
	done chan struct{}
}

func (l *idleListener) Accept() (net.Conn, error) { <-l.done; return nil, net.ErrClosed }
func (l *idleListener) Close() error              { l.once.Do(func() { close(l.done) }); return nil }
func (l *idleListener) Addr() net.Addr            { return &net.TCPAddr{} }

func TestRunShouldListenAndShutdownCleanly(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
	l.Close()

	cfg := &config.Config{ServerPort: port}
	srv, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Run(ctx, srv) }()

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:" + port + "/health")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 10*time.Millisecond)

	cancel()
	require.NoError(t, <-done)
}

func TestServeShouldShutdownCleanlyWhenContextCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := &config.Config{ServerPort: "8080"}
		srv, err := New(cfg)
		require.NoError(t, err)

		l := &idleListener{done: make(chan struct{})}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		done := make(chan error, 1)
		go func() { done <- serve(ctx, srv, l) }()

		synctest.Wait() // goroutine durably blocked on l.Accept()
		cancel()        // trigger shutdown goroutine
		synctest.Wait() // Shutdown calls l.Close() → Accept unblocks → serve exits

		require.NoError(t, <-done)
	})
}
