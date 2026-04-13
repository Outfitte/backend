package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/outfitte/backend/internal/domain"
)

type tokenLogout interface {
	Logout(ctx context.Context, refreshToken string) error
}

type userRegistrar interface {
	Register(ctx context.Context, username, password string) (domain.User, error)
}

type tokenIssuer interface {
	Login(ctx context.Context, username, password string) (accessToken, refreshToken string, user domain.User, err error)
}

type tokenRefresher interface {
	Refresh(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, user domain.User, err error)
}

// AuthHandler handles POST /auth/register.
type AuthHandler struct {
	users   userRegistrar
	auth    tokenIssuer
	refresh tokenRefresher
	logout  tokenLogout
	log     *slog.Logger
}

// NewAuthHandler creates an AuthHandler with a logger pre-scoped to handler=auth.
func NewAuthHandler(users userRegistrar, auth tokenIssuer, refresh tokenRefresher, logout tokenLogout, log *slog.Logger) *AuthHandler {
	return &AuthHandler{users: users, auth: auth, refresh: refresh, logout: logout, log: log.With("handler", "auth")}
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Register")
	log.InfoContext(ctx, "started")

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := h.users.Register(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "username already taken"})
			return
		}
		log.ErrorContext(ctx, "register failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	accessToken, refreshToken, _, err := h.auth.Login(ctx, req.Username, req.Password)
	if err != nil {
		log.ErrorContext(ctx, "login after register failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "user_id", user.GetID())
	writeJSON(w, http.StatusCreated, registerResponse{
		User: userResponse{
			ID:        user.GetID(),
			Email:     user.Email,
			Role:      string(user.Role),
			CreatedAt: user.CreatedAt,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Login")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	accessToken, refreshToken, user, err := h.auth.Login(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		log.ErrorContext(ctx, "login failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "user_id", user.GetID())
	writeJSON(w, http.StatusOK, loginResponse{
		User: userResponse{
			ID:        user.GetID(),
			Email:     user.Email,
			Role:      string(user.Role),
			CreatedAt: user.CreatedAt,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// Refresh handles POST /auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Refresh")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	accessToken, newRefreshToken, user, err := h.refresh.Refresh(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) || errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrSessionExpired) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired refresh token"})
			return
		}
		log.ErrorContext(ctx, "refresh failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "user_id", user.GetID())
	writeJSON(w, http.StatusOK, refreshResponse{
		User: userResponse{
			ID:        user.GetID(),
			Email:     user.Email,
			Role:      string(user.Role),
			CreatedAt: user.CreatedAt,
		},
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handles POST /auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Logout")
	log.InfoContext(ctx, "started")

	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.logout.Logout(ctx, req.RefreshToken); err != nil {
		if errors.Is(err, domain.ErrUnauthorized) || errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		log.ErrorContext(ctx, "logout failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded")
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
