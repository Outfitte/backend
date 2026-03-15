package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/outfitte/outfitte/internal/domain"
)

type mediaProvider interface {
	Download(ctx context.Context, key string) (io.ReadCloser, error)
}

// MediaHandler handles GET /media/{key...}.
type MediaHandler struct {
	media mediaProvider
	log   *slog.Logger
}

// NewMediaHandler creates a MediaHandler with a logger pre-scoped to handler=media.
func NewMediaHandler(media mediaProvider, log *slog.Logger) *MediaHandler {
	return &MediaHandler{media: media, log: log.With("handler", "media")}
}

// Download handles GET /media/{key...}.
func (h *MediaHandler) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Download")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	key := r.PathValue("key")

	rc, err := h.media.Download(ctx, key)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		log.ErrorContext(ctx, "download failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	defer rc.Close()

	if ct := mime.TypeByExtension(filepath.Ext(key)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	log.InfoContext(ctx, "succeeded", "key", key)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}
