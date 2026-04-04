package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/outfitte/backend/internal/domain"
)

type userLister interface {
	List(ctx context.Context) ([]domain.User, error)
}

// UserHandler handles user-related endpoints.
type UserHandler struct {
	users userLister
	log   *slog.Logger
}

// NewUserHandler creates a UserHandler with a logger pre-scoped to handler=user.
func NewUserHandler(users userLister, log *slog.Logger) *UserHandler {
	return &UserHandler{users: users, log: log.With("handler", "user")}
}

// List handles GET /users.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
}
