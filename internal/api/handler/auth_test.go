package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/outfitte/backend/internal/api/handler"
	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeUserRegistrar struct {
	registerFn func(ctx context.Context, username, password string) (domain.User, error)
}

func (f *fakeUserRegistrar) Register(ctx context.Context, username, password string) (domain.User, error) {
	return f.registerFn(ctx, username, password)
}

type fakeTokenIssuer struct {
	loginFn func(ctx context.Context, username, password string) (string, string, error)
}

func (f *fakeTokenIssuer) Login(ctx context.Context, username, password string) (string, string, error) {
	return f.loginFn(ctx, username, password)
}

type fakeTokenRefresher struct {
	refreshFn func(ctx context.Context, refreshToken string) (string, string, error)
}

func (f *fakeTokenRefresher) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	return f.refreshFn(ctx, refreshToken)
}

type fakeTokenLogout struct {
	logoutFn func(ctx context.Context, refreshToken string) error
}

func (f *fakeTokenLogout) Logout(ctx context.Context, refreshToken string) error {
	return f.logoutFn(ctx, refreshToken)
}

// --- helpers ---

func newAuthHandler(users *fakeUserRegistrar, auth *fakeTokenIssuer) *handler.AuthHandler {
	return handler.NewAuthHandler(users, auth, &fakeTokenRefresher{}, &fakeTokenLogout{}, slog.New(slog.DiscardHandler))
}

func postLogout(t *testing.T, h *handler.AuthHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Logout(w, req)
	return w
}

func postRefresh(t *testing.T, h *handler.AuthHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Refresh(w, req)
	return w
}

func postRegister(t *testing.T, h *handler.AuthHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, req)
	return w
}

// --- tests ---

func TestRegisterHandlerShouldReturn409WhenUsernameAlreadyTaken(t *testing.T) {
	users := &fakeUserRegistrar{
		registerFn: func(_ context.Context, _, _ string) (domain.User, error) {
			return domain.User{}, domain.ErrConflict
		},
	}
	auth := &fakeTokenIssuer{}
	h := newAuthHandler(users, auth)

	w := postRegister(t, h, `{"username":"alice@example.com","password":"secret"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "username already taken", body["error"])
}

func TestRegisterHandlerShouldReturn403WhenRegistrationIsDisabled(t *testing.T) {
	users := &fakeUserRegistrar{
		registerFn: func(_ context.Context, _, _ string) (domain.User, error) {
			return domain.User{}, domain.ErrRegistrationDisabled
		},
	}
	auth := &fakeTokenIssuer{}
	h := newAuthHandler(users, auth)

	w := postRegister(t, h, `{"username":"alice@example.com","password":"secret"}`)

	require.Equal(t, http.StatusForbidden, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "registration is disabled", body["error"])
}

func TestRegisterHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	users := &fakeUserRegistrar{
		registerFn: func(_ context.Context, _, _ string) (domain.User, error) {
			return domain.User{}, domain.ErrIO
		},
	}
	auth := &fakeTokenIssuer{}
	h := newAuthHandler(users, auth)

	w := postRegister(t, h, `{"username":"alice@example.com","password":"secret"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "internal server error", body["error"])
}

func TestRegisterHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{}
	h := newAuthHandler(users, auth)

	w := postRegister(t, h, `not-json`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "invalid request body", body["error"])
}

func TestRegisterHandlerShouldReturn201WhenRegistrationSucceeds(t *testing.T) {
	var registeredUser domain.User
	registeredUser.ID = "user-42"
	registeredUser.Email = "alice@example.com"
	registeredUser.Role = domain.RoleMember

	users := &fakeUserRegistrar{
		registerFn: func(_ context.Context, _, _ string) (domain.User, error) {
			return registeredUser, nil
		},
	}
	auth := &fakeTokenIssuer{
		loginFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "access-tok", "refresh-tok", nil
		},
	}
	h := newAuthHandler(users, auth)

	w := postRegister(t, h, `{"username":"alice@example.com","password":"secret"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	userObj, ok := body["user"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "user-42", userObj["id"])
	require.Equal(t, "alice@example.com", userObj["email"])
	require.Equal(t, "member", userObj["role"])
	require.Equal(t, "access-tok", body["access_token"])
	require.Equal(t, "refresh-tok", body["refresh_token"])
}

func postLogin(t *testing.T, h *handler.AuthHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, req)
	return w
}

func TestLoginHandlerShouldReturn401WhenCredentialsAreInvalid(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{
		loginFn: func(ctx context.Context, _, _ string) (string, string, error) {
			return "", "", domain.ErrUnauthorized
		},
	}
	h := newAuthHandler(users, auth)

	w := postLogin(t, h, `{"username":"alice","password":"wrong"}`)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "invalid credentials", body["error"])
}

func TestLoginHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{}
	h := newAuthHandler(users, auth)

	w := postLogin(t, h, `not-json`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "invalid request body", body["error"])
}

func TestLoginHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{
		loginFn: func(ctx context.Context, _, _ string) (string, string, error) {
			return "", "", domain.ErrIO
		},
	}
	h := newAuthHandler(users, auth)

	w := postLogin(t, h, `{"username":"alice","password":"secret"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "internal server error", body["error"])
}

func TestLoginHandlerShouldReturn200WhenLoginSucceeds(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{
		loginFn: func(ctx context.Context, _, _ string) (string, string, error) {
			return "access-tok", "refresh-tok", nil
		},
	}
	h := newAuthHandler(users, auth)

	w := postLogin(t, h, `{"username":"alice","password":"secret"}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "access-tok", body["access_token"])
	require.Equal(t, "refresh-tok", body["refresh_token"])
}

func TestRefreshHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{}
	refresher := &fakeTokenRefresher{}
	h := handler.NewAuthHandler(users, auth, refresher, &fakeTokenLogout{}, slog.New(slog.DiscardHandler))

	w := postRefresh(t, h, `not-json`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "invalid request body", body["error"])
}

func TestRefreshHandlerShouldReturn401WhenTokenIsInvalidOrExpired(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{}
	refresher := &fakeTokenRefresher{
		refreshFn: func(ctx context.Context, _ string) (string, string, error) {
			return "", "", domain.ErrUnauthorized
		},
	}
	h := handler.NewAuthHandler(users, auth, refresher, &fakeTokenLogout{}, slog.New(slog.DiscardHandler))

	w := postRefresh(t, h, `{"refresh_token":"old-tok"}`)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "invalid or expired refresh token", body["error"])
}

func TestRefreshHandlerShouldReturn401WhenTokenNotFound(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{}
	refresher := &fakeTokenRefresher{
		refreshFn: func(ctx context.Context, _ string) (string, string, error) {
			return "", "", domain.ErrNotFound
		},
	}
	h := handler.NewAuthHandler(users, auth, refresher, &fakeTokenLogout{}, slog.New(slog.DiscardHandler))

	w := postRefresh(t, h, `{"refresh_token":"deleted-tok"}`)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "invalid or expired refresh token", body["error"])
}

func TestRefreshHandlerShouldReturn401WhenSessionExpired(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{}
	refresher := &fakeTokenRefresher{
		refreshFn: func(ctx context.Context, _ string) (string, string, error) {
			return "", "", domain.ErrSessionExpired
		},
	}
	h := handler.NewAuthHandler(users, auth, refresher, &fakeTokenLogout{}, slog.New(slog.DiscardHandler))

	w := postRefresh(t, h, `{"refresh_token":"expired-tok"}`)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "invalid or expired refresh token", body["error"])
}

func TestRefreshHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{}
	refresher := &fakeTokenRefresher{
		refreshFn: func(ctx context.Context, _ string) (string, string, error) {
			return "", "", domain.ErrIO
		},
	}
	h := handler.NewAuthHandler(users, auth, refresher, &fakeTokenLogout{}, slog.New(slog.DiscardHandler))

	w := postRefresh(t, h, `{"refresh_token":"old-tok"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "internal server error", body["error"])
}

func TestRefreshHandlerShouldReturn200WhenRefreshSucceeds(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{}
	refresher := &fakeTokenRefresher{
		refreshFn: func(ctx context.Context, _ string) (string, string, error) {
			return "new-access-tok", "new-refresh-tok", nil
		},
	}
	h := handler.NewAuthHandler(users, auth, refresher, &fakeTokenLogout{}, slog.New(slog.DiscardHandler))

	w := postRefresh(t, h, `{"refresh_token":"old-tok"}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "new-access-tok", body["access_token"])
	require.Equal(t, "new-refresh-tok", body["refresh_token"])
}

func TestLogoutHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := handler.NewAuthHandler(&fakeUserRegistrar{}, &fakeTokenIssuer{}, &fakeTokenRefresher{}, &fakeTokenLogout{}, slog.New(slog.DiscardHandler))

	w := postLogout(t, h, `not-json`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "invalid request body", body["error"])
}

func TestLogoutHandlerShouldReturn401WhenServiceReturnsUnauthorized(t *testing.T) {
	logout := &fakeTokenLogout{
		logoutFn: func(_ context.Context, _ string) error {
			return domain.ErrUnauthorized
		},
	}
	h := handler.NewAuthHandler(&fakeUserRegistrar{}, &fakeTokenIssuer{}, &fakeTokenRefresher{}, logout, slog.New(slog.DiscardHandler))

	w := postLogout(t, h, `{"refresh_token":"tok"}`)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "unauthorized", body["error"])
}

func TestLogoutHandlerShouldReturn401WhenTokenNotFound(t *testing.T) {
	logout := &fakeTokenLogout{
		logoutFn: func(_ context.Context, _ string) error {
			return domain.ErrNotFound
		},
	}
	h := handler.NewAuthHandler(&fakeUserRegistrar{}, &fakeTokenIssuer{}, &fakeTokenRefresher{}, logout, slog.New(slog.DiscardHandler))

	w := postLogout(t, h, `{"refresh_token":"deleted-tok"}`)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "unauthorized", body["error"])
}

func TestLogoutHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	logout := &fakeTokenLogout{
		logoutFn: func(_ context.Context, _ string) error {
			return domain.ErrIO
		},
	}
	h := handler.NewAuthHandler(&fakeUserRegistrar{}, &fakeTokenIssuer{}, &fakeTokenRefresher{}, logout, slog.New(slog.DiscardHandler))

	w := postLogout(t, h, `{"refresh_token":"tok"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "internal server error", body["error"])
}

func TestLogoutHandlerShouldReturn204WhenLogoutSucceeds(t *testing.T) {
	logout := &fakeTokenLogout{
		logoutFn: func(_ context.Context, _ string) error {
			return nil
		},
	}
	h := handler.NewAuthHandler(&fakeUserRegistrar{}, &fakeTokenIssuer{}, &fakeTokenRefresher{}, logout, slog.New(slog.DiscardHandler))

	w := postLogout(t, h, `{"refresh_token":"tok"}`)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Empty(t, w.Body.String())
}

func TestAuthCycleShouldSucceedWhenLoginRefreshAndLogout(t *testing.T) {
	var logoutReceivedToken string

	auth := &fakeTokenIssuer{
		loginFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "access-tok-1", "refresh-tok-1", nil
		},
	}
	refresher := &fakeTokenRefresher{
		refreshFn: func(_ context.Context, refreshToken string) (string, string, error) {
			require.Equal(t, "refresh-tok-1", refreshToken)
			return "access-tok-2", "refresh-tok-2", nil
		},
	}
	logout := &fakeTokenLogout{
		logoutFn: func(_ context.Context, refreshToken string) error {
			logoutReceivedToken = refreshToken
			return nil
		},
	}
	h := handler.NewAuthHandler(&fakeUserRegistrar{}, auth, refresher, logout, slog.New(slog.DiscardHandler))

	// Step 1: Login
	w := postLogin(t, h, `{"username":"alice","password":"secret"}`)
	require.Equal(t, http.StatusOK, w.Code)
	var loginBody map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&loginBody))
	require.Equal(t, "access-tok-1", loginBody["access_token"])
	require.Equal(t, "refresh-tok-1", loginBody["refresh_token"])

	// Step 2: Refresh using the token from login
	w = postRefresh(t, h, fmt.Sprintf(`{"refresh_token":%q}`, loginBody["refresh_token"]))
	require.Equal(t, http.StatusOK, w.Code)
	var refreshBody map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&refreshBody))
	require.Equal(t, "access-tok-2", refreshBody["access_token"])
	require.Equal(t, "refresh-tok-2", refreshBody["refresh_token"])

	// Step 3: Logout using the rotated token from refresh
	w = postLogout(t, h, fmt.Sprintf(`{"refresh_token":%q}`, refreshBody["refresh_token"]))
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "refresh-tok-2", logoutReceivedToken)
}

func TestRegisterHandlerShouldReturn500WhenTokenIssuanceFails(t *testing.T) {
	var registeredUser domain.User
	registeredUser.ID = "user-42"
	registeredUser.Email = "alice@example.com"
	registeredUser.Role = domain.RoleMember

	users := &fakeUserRegistrar{
		registerFn: func(_ context.Context, _, _ string) (domain.User, error) {
			return registeredUser, nil
		},
	}
	auth := &fakeTokenIssuer{
		loginFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", domain.ErrIO
		},
	}
	h := newAuthHandler(users, auth)

	w := postRegister(t, h, `{"username":"alice@example.com","password":"secret"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "internal server error", body["error"])
}
