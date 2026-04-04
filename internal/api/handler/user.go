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
	ctx := r.Context()
	log := h.log.With("call", "List")
	log.InfoContext(ctx, "started")

	users, err := h.users.List(ctx)
	if err != nil {
		log.ErrorContext(ctx, "list failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = userResponse{
			ID:        u.GetID(),
			Email:     u.Email,
			Role:      string(u.Role),
			CreatedAt: u.CreatedAt,
		}
	}

	log.InfoContext(ctx, "succeeded", "count", len(users))
	writeJSON(w, http.StatusOK, resp)
}
