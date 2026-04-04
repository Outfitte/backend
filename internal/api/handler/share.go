package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/service"
)

type shareService interface {
	Create(ctx context.Context, callerID string, input service.CreateShareInput) (domain.Share, error)
	ListOutgoing(ctx context.Context, callerID string) ([]service.ShareView, error)
	ListSharedWithMe(ctx context.Context, callerID string) (service.SharedWithMeResult, error)
	Revoke(ctx context.Context, callerID, shareID string) error
}

// ShareHandler handles share-related HTTP endpoints.
type ShareHandler struct {
	shares shareService
	log    *slog.Logger
}

// NewShareHandler creates a ShareHandler with a logger pre-scoped to handler=share.
func NewShareHandler(shares shareService, log *slog.Logger) *ShareHandler {
	return &ShareHandler{shares: shares, log: log.With("handler", "share")}
}

// Create handles POST /shares.
func (h *ShareHandler) Create(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("not implemented").Error(), http.StatusInternalServerError)
}

// ListOutgoing handles GET /shares.
func (h *ShareHandler) ListOutgoing(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("not implemented").Error(), http.StatusInternalServerError)
}

// ListSharedWithMe handles GET /shares/with-me.
func (h *ShareHandler) ListSharedWithMe(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("not implemented").Error(), http.StatusInternalServerError)
}

// Revoke handles DELETE /shares/{id}.
func (h *ShareHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	http.Error(w, errors.New("not implemented").Error(), http.StatusInternalServerError)
}
