package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/outfitte/backend/internal/api/handler"
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

// --- helpers ---

func newUserHandler(lister *fakeUserLister) *handler.UserHandler {
	return handler.NewUserHandler(lister, slog.New(slog.DiscardHandler))
}

func getUsers(t *testing.T, h *handler.UserHandler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	h.List(w, req)
	return w
}

// --- tests ---

func TestUserListHandlerShouldReturn503WhenContextCancelled(t *testing.T) {
	lister := &fakeUserLister{
		listFn: func(_ context.Context) ([]domain.User, error) {
			return nil, nil
		},
	}
	h := newUserHandler(lister)

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
	h := newUserHandler(lister)

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
	h := newUserHandler(lister)

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
	h := newUserHandler(lister)

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
