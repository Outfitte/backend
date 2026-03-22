package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/outfitte/outfitte/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestNewShouldServeHealthWhenGivenValidConfig(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	srv := New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil)
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
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	cfg := &config.Config{ServerPort: strconv.Itoa(port)}
	srv := New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Run(ctx, srv) }()

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:" + strconv.Itoa(port) + "/health")
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
func (l *errorListener) Close() error   { return nil }
func (l *errorListener) Addr() net.Addr { return &net.TCPAddr{} }

func TestServeShouldReturnErrorWhenListenerFails(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	srv := New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil)

	l := &errorListener{done: make(chan struct{})}
	err := serve(t.Context(), srv, l)
	require.ErrorIs(t, err, errInjected)
}

func TestServeShouldShutdownCleanlyWhenContextCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := &config.Config{ServerPort: "8080"}
		srv := New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil)

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

func getURL(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func patchJSON(t *testing.T, srv *httptest.Server, path, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPatch, srv.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func deleteURL(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, srv.URL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestNewShouldReturn401WhenGetLocationsCalledWithoutAuth(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := getURL(t, ts, "/locations")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestNewShouldReturn401WhenPostLocationsCalledWithoutAuth(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := postJSON(t, ts, "/locations", `{"label":"closet"}`)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestNewShouldReturn401WhenGetLocationByIDCalledWithoutAuth(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := getURL(t, ts, "/locations/loc-1")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestNewShouldReturn401WhenPatchLocationCalledWithoutAuth(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := patchJSON(t, ts, "/locations/loc-1", `{"label":"shelf"}`)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestNewShouldReturn401WhenDeleteLocationCalledWithoutAuth(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := deleteURL(t, ts, "/locations/loc-1")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestNewShouldReturn401WhenMoveLocationCalledWithoutAuth(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := patchJSON(t, ts, "/locations/loc-1/move", `{"parent_id":null}`)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func postJSON(t *testing.T, srv *httptest.Server, path, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestNewShouldReturn400WhenRegisterCalledWithInvalidBody(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := postJSON(t, ts, "/auth/register", "not-json")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestNewShouldReturn400WhenLoginCalledWithInvalidBody(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := postJSON(t, ts, "/auth/login", "not-json")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestNewShouldReturn400WhenRefreshCalledWithInvalidBody(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := postJSON(t, ts, "/auth/refresh", "not-json")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestNewShouldReturn400WhenLogoutCalledWithInvalidBody(t *testing.T) {
	cfg := &config.Config{ServerPort: "8080"}
	ts := httptest.NewServer(New(cfg, discardLogger(), nil, nil, nil, nil, nil, nil).Handler)
	defer ts.Close()

	resp := postJSON(t, ts, "/auth/logout", "not-json")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
