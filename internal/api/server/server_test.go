package server

import (
	"context"
	"errors"
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
	srv := New(cfg)
	assert.Equal(t, ":8080", srv.Addr)

	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRunShouldReturnErrorWhenAddressInvalid(t *testing.T) {
	srv := &http.Server{Addr: ":::not-a-port"}
	err := Run(t.Context(), srv)
	require.Error(t, err)
	assert.ErrorContains(t, err, "listen")
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
	srv := New(cfg)

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

// errorListener returns an injected error on the first Accept, then blocks.
type errorListener struct {
	once sync.Once
	done chan struct{}
}

var errInjected = errors.New("injected accept error")

func (l *errorListener) Accept() (net.Conn, error) {
	var fired bool
	l.once.Do(func() { fired = true })
	if fired {
		return nil, errInjected
	}
	<-l.done
	return nil, net.ErrClosed
}
func (l *errorListener) Close() error  { return nil }
func (l *errorListener) Addr() net.Addr { return &net.TCPAddr{} }

func TestServeShouldReturnErrorWhenListenerFails(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	srv := New(cfg)

	l := &errorListener{done: make(chan struct{})}
	err := serve(t.Context(), srv, l)
	require.ErrorIs(t, err, errInjected)
}

func TestServeShouldShutdownCleanlyWhenContextCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := &config.Config{ServerPort: "8080"}
		srv := New(cfg)

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
