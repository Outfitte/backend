package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/outfitte/backend/internal/api/handler"
	"github.com/outfitte/backend/internal/api/middleware"
	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeUserLister struct {
	listFn func(ctx context.Context) ([]domain.User, error)
}

func (f *fakeUserLister) List(ctx context.Context) ([]domain.User, error) {
	return f.listFn(ctx)
}

type fakeUserGetter struct {
	getByIDFn func(ctx context.Context, id string) (domain.User, error)
}

func (f *fakeUserGetter) GetByID(ctx context.Context, id string) (domain.User, error) {
	return f.getByIDFn(ctx, id)
}

// --- helpers ---

func newUserHandler(lister *fakeUserLister, getter *fakeUserGetter) *handler.UserHandler {
	return handler.NewUserHandler(lister, getter, slog.New(slog.DiscardHandler))
}

func getUsers(t *testing.T, h *handler.UserHandler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	h.List(w, req)
	return w
}

// --- tests ---

// --- Me handler tests ---

func meRequest(t *testing.T, h *handler.UserHandler, ctx context.Context) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, req)
	return w
}

func TestMeHandlerShouldReturn503WhenContextCancelled(t *testing.T) {
	h := newUserHandler(&fakeUserLister{}, &fakeUserGetter{})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	w := meRequest(t, h, ctx)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "request cancelled", body["error"])
}

func meRequestWithAuth(t *testing.T, h *handler.UserHandler, userID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), userID)
	return meRequest(t, h, ctx)
}

func TestMeHandlerShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newUserHandler(&fakeUserLister{}, &fakeUserGetter{})

	// No auth middleware → no user ID in context.
	w := meRequest(t, h, t.Context())

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "internal server error", body["error"])
}

func TestMeHandlerShouldReturn500WhenGetByIDFails(t *testing.T) {
	getter := &fakeUserGetter{
		getByIDFn: func(_ context.Context, _ string) (domain.User, error) {
			return domain.User{}, domain.ErrIO
		},
	}
	h := newUserHandler(&fakeUserLister{}, getter)

	w := meRequestWithAuth(t, h, "user-42")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "internal server error", body["error"])
}

func TestMeHandlerShouldReturn200WithProfileWhenAuthenticated(t *testing.T) {
	var u domain.User
	u.ID = "user-42"
	u.Email = "alice@example.com"
	u.Role = domain.RoleAdmin

	getter := &fakeUserGetter{
		getByIDFn: func(_ context.Context, id string) (domain.User, error) {
			require.Equal(t, "user-42", id)
			return u, nil
		},
	}
	h := newUserHandler(&fakeUserLister{}, getter)

	w := meRequestWithAuth(t, h, "user-42")

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "user-42", body["id"])
	require.Equal(t, "alice@example.com", body["email"])
	require.Equal(t, "admin", body["role"])
	require.NotEmpty(t, body["created_at"])
}

// --- List handler tests ---

func TestUserListHandlerShouldReturn503WhenContextCancelled(t *testing.T) {
	lister := &fakeUserLister{
		listFn: func(_ context.Context) ([]domain.User, error) {
			return nil, nil
		},
	}
	h := newUserHandler(lister, &fakeUserGetter{})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "request cancelled", body["error"])
}

func TestUserListHandlerShouldReturn500WhenRepositoryFails(t *testing.T) {
	lister := &fakeUserLister{
		listFn: func(_ context.Context) ([]domain.User, error) {
			return nil, domain.ErrIO
		},
	}
	h := newUserHandler(lister, &fakeUserGetter{})

	w := getUsers(t, h)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "internal server error", body["error"])
}

func TestUserListHandlerShouldReturn200WithEmptyArrayWhenNoUsers(t *testing.T) {
	lister := &fakeUserLister{
		listFn: func(_ context.Context) ([]domain.User, error) {
			return []domain.User{}, nil
		},
	}
	h := newUserHandler(lister, &fakeUserGetter{})

	w := getUsers(t, h)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Empty(t, body)
}

func TestUserListHandlerShouldReturn200WithUsersWhenUsersExist(t *testing.T) {
	var u1, u2 domain.User
	u1.ID = "user-1"
	u1.Email = "alice@example.com"
	u1.Role = domain.RoleAdmin

	u2.ID = "user-2"
	u2.Email = "bob@example.com"
	u2.Role = domain.RoleMember

	lister := &fakeUserLister{
		listFn: func(_ context.Context) ([]domain.User, error) {
			return []domain.User{u1, u2}, nil
		},
	}
	h := newUserHandler(lister, &fakeUserGetter{})

	w := getUsers(t, h)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body, 2)

	require.Equal(t, "user-1", body[0]["id"])
	require.Equal(t, "alice@example.com", body[0]["email"])
	require.Nil(t, body[0]["role"])
	require.Nil(t, body[0]["created_at"])

	require.Equal(t, "user-2", body[1]["id"])
	require.Equal(t, "bob@example.com", body[1]["email"])
	require.Nil(t, body[1]["role"])
	require.Nil(t, body[1]["created_at"])
}
