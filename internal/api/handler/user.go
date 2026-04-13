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

type userGetter interface {
	GetByID(ctx context.Context, id string) (domain.User, error)
}

// userSummaryResponse is the minimal user representation used for share recipient selection.
type userSummaryResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// UserHandler handles user-related endpoints.
type UserHandler struct {
	users  userLister
	getter userGetter
	log    *slog.Logger
}

// NewUserHandler creates a UserHandler with a logger pre-scoped to handler=user.
func NewUserHandler(users userLister, getter userGetter, log *slog.Logger) *UserHandler {
	return &UserHandler{users: users, getter: getter, log: log.With("handler", "user")}
}

// Me handles GET /users/me.
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Me")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	userID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	user, err := h.getter.GetByID(ctx, userID)
	if err != nil {
		log.ErrorContext(ctx, "get failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "user_id", userID)
	writeJSON(w, http.StatusOK, userResponse{
		ID:        user.GetID(),
		Email:     user.Email,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt,
	})
}

// List handles GET /users.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "List")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	users, err := h.users.List(ctx)
	if err != nil {
		log.ErrorContext(ctx, "list failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := make([]userSummaryResponse, len(users))
	for i, u := range users {
		resp[i] = userSummaryResponse{
			ID:    u.GetID(),
			Email: u.Email,
		}
	}

	log.InfoContext(ctx, "succeeded", "count", len(users))
	writeJSON(w, http.StatusOK, resp)
}
