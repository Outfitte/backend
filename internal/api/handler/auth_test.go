package handler_test

import (
	"context"
	"encoding/json"
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

// --- helpers ---

func newAuthHandler(users *fakeUserRegistrar, auth *fakeTokenIssuer) *handler.AuthHandler {
	return handler.NewAuthHandler(users, auth, slog.New(slog.DiscardHandler))
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
		loginFn: func(_ context.Context, _, _ string) (string, string, error) {
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

func TestLoginHandlerShouldReturn200WhenLoginSucceeds(t *testing.T) {
	users := &fakeUserRegistrar{}
	auth := &fakeTokenIssuer{
		loginFn: func(_ context.Context, _, _ string) (string, string, error) {
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
		loginFn: func(_ context.Context, _, _ string) (string, string, error) {
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
