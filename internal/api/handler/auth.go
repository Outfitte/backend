package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
)

type userRegistrar interface {
	Register(ctx context.Context, username, password string) (domain.User, error)
}

type tokenIssuer interface {
	Login(ctx context.Context, username, password string) (accessToken, refreshToken string, err error)
}

type tokenRefresher interface {
	Refresh(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, err error)
}

// AuthHandler handles POST /auth/register.
type AuthHandler struct {
	users   userRegistrar
	auth    tokenIssuer
	refresh tokenRefresher
	log     *slog.Logger
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(users userRegistrar, auth tokenIssuer, refresh tokenRefresher, log *slog.Logger) *AuthHandler {
	return &AuthHandler{users: users, auth: auth, refresh: refresh, log: log}
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type registerResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.log.InfoContext(ctx, "register called")

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
		h.log.ErrorContext(ctx, "register failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	accessToken, refreshToken, err := h.auth.Login(ctx, req.Username, req.Password)
	if err != nil {
		h.log.ErrorContext(ctx, "login after register failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	h.log.InfoContext(ctx, "register succeeded", "user_id", user.GetID())
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
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.log.InfoContext(ctx, "login called")

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	accessToken, refreshToken, err := h.auth.Login(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		h.log.ErrorContext(ctx, "login failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	h.log.InfoContext(ctx, "login succeeded")
	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles POST /auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.log.InfoContext(ctx, "refresh called")

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	accessToken, newRefreshToken, err := h.refresh.Refresh(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired refresh token"})
			return
		}
		h.log.ErrorContext(ctx, "refresh failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	h.log.InfoContext(ctx, "refresh succeeded")
	writeJSON(w, http.StatusOK, refreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
