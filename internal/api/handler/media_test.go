package handler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeMediaProvider struct {
	downloadFn func(ctx context.Context, key string) (io.ReadCloser, error)
}

func (f *fakeMediaProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return f.downloadFn(ctx, key)
}

// --- helpers ---

func getMedia(t *testing.T, h *handler.MediaHandler, key string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/media/"+key, nil)
	req.SetPathValue("key", key)
	w := httptest.NewRecorder()
	h.Download(w, req)
	return w
}

// --- tests ---

func TestMediaDownloadShouldReturn404WhenKeyNotFound(t *testing.T) {
	mp := &fakeMediaProvider{
		downloadFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewMediaHandler(mp, slog.New(slog.DiscardHandler))

	w := getMedia(t, h, "abc123")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMediaDownloadShouldReturn500WhenProviderFails(t *testing.T) {
	mp := &fakeMediaProvider{
		downloadFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return nil, domain.ErrIO
		},
	}
	h := handler.NewMediaHandler(mp, slog.New(slog.DiscardHandler))

	w := getMedia(t, h, "abc123")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMediaDownloadShouldReturn200WithBodyWhenKeyExists(t *testing.T) {
	content := "fake image bytes"
	mp := &fakeMediaProvider{
		downloadFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(content)), nil
		},
	}
	h := handler.NewMediaHandler(mp, slog.New(slog.DiscardHandler))

	w := getMedia(t, h, "abc123.jpg")

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, content, w.Body.String())
}

func TestMediaDownloadShouldSetContentTypeFromExtensionWhenExtensionIsKnown(t *testing.T) {
	mp := &fakeMediaProvider{
		downloadFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("data")), nil
		},
	}
	h := handler.NewMediaHandler(mp, slog.New(slog.DiscardHandler))

	w := getMedia(t, h, "photo.png")

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "image/png", w.Header().Get("Content-Type"))
}

func TestMediaDownloadShouldNotSetContentTypeWhenExtensionIsUnknown(t *testing.T) {
	mp := &fakeMediaProvider{
		downloadFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("data")), nil
		},
	}
	h := handler.NewMediaHandler(mp, slog.New(slog.DiscardHandler))

	w := getMedia(t, h, "abc123")

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Header().Get("Content-Type"))
}

func TestMediaDownloadShouldReturn500WhenProviderReturnsWrappedError(t *testing.T) {
	wrapped := errors.New("some internal error")
	mp := &fakeMediaProvider{
		downloadFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return nil, wrapped
		},
	}
	h := handler.NewMediaHandler(mp, slog.New(slog.DiscardHandler))

	w := getMedia(t, h, "abc123")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
