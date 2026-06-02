package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/adapter/store/json"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

func TestNewUserRepositoryShouldImplementUserRepository(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	require.Implements(t, (*ports.UserRepository)(nil), r)
}

func TestUserGetShouldReturnNotFoundWhenUserDoesNotExist(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())

	_, err := r.Get(t.Context(), "u1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserGetShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Get(ctx, "u1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserSaveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.User{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserSaveShouldReturnErrConflictWhenEmailUsedByDifferentUser(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	var u1, u2 domain.User
	u1.ID = "u1"
	u1.Email = "alice@example.com"
	u2.ID = "u2"
	u2.Email = "alice@example.com"
	require.NoError(t, r.Save(t.Context(), u1))

	err := r.Save(t.Context(), u2)
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestUserSaveShouldAllowUpsertOfSameUser(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	var u domain.User
	u.ID = "u1"
	u.Email = "alice@example.com"
	require.NoError(t, r.Save(t.Context(), u))

	u.PasswordHash = "newhash"
	require.NoError(t, r.Save(t.Context(), u))
}

func TestUserGetShouldReturnUserWhenFound(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	var u domain.User
	u.ID = "u1"
	u.Email = "alice@example.com"
	require.NoError(t, r.Save(t.Context(), u))

	got, err := r.Get(t.Context(), "u1")
	require.NoError(t, err)
	require.Equal(t, u, got)
}

func TestUserGetByEmailShouldReturnNotFoundWhenNoUserExists(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())

	_, err := r.GetByEmail(t.Context(), "alice@example.com")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserGetByEmailShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.GetByEmail(ctx, "alice@example.com")
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserGetByEmailShouldReturnUserWhenFound(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	var u domain.User
	u.ID = "u1"
	u.Email = "alice@example.com"
	require.NoError(t, r.Save(t.Context(), u))

	got, err := r.GetByEmail(t.Context(), "alice@example.com")
	require.NoError(t, err)
	require.Equal(t, u, got)
}

func TestUserCountShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Count(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserCountShouldReturnZeroWhenNoUsersExist(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())

	count, err := r.Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestUserCountShouldReturnCorrectCount(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	var u1, u2 domain.User
	u1.ID = "u1"
	u1.Email = "alice@example.com"
	u2.ID = "u2"
	u2.Email = "bob@example.com"
	require.NoError(t, r.Save(t.Context(), u1))
	require.NoError(t, r.Save(t.Context(), u2))

	count, err := r.Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestUserListShouldReturnEmptyWhenNoUsersExist(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())

	users, err := r.List(t.Context())
	require.NoError(t, err)
	require.Empty(t, users)
}

func TestUserListShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.List(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserSaveShouldReturnErrIOWhenDataFileIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte("not json"), 0o644))
	r := json.NewUserRepository(dir)

	var u domain.User
	u.ID = "u1"
	err := r.Save(t.Context(), u)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUserGetByEmailShouldReturnErrIOWhenDataFileIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte("not json"), 0o644))
	r := json.NewUserRepository(dir)

	_, err := r.GetByEmail(t.Context(), "alice@example.com")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUserCountShouldReturnErrIOWhenDataFileIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte("not json"), 0o644))
	r := json.NewUserRepository(dir)

	_, err := r.Count(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUserListShouldReturnAllUsers(t *testing.T) {
	r := json.NewUserRepository(t.TempDir())
	var u1, u2 domain.User
	u1.ID = "u1"
	u1.Email = "alice@example.com"
	u2.ID = "u2"
	u2.Email = "bob@example.com"
	require.NoError(t, r.Save(t.Context(), u1))
	require.NoError(t, r.Save(t.Context(), u2))

	users, err := r.List(t.Context())
	require.NoError(t, err)
	require.Len(t, users, 2)
}
