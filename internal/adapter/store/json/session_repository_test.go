package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewSessionRepositoryShouldImplementSessionRepository(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	require.Implements(t, (*ports.SessionRepository)(nil), r)
}

func TestGetShouldReturnNotFoundWhenSessionDoesNotExist(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())

	_, err := r.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetShouldReturnSessionWhenFound(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	var s domain.Session
	s.ID = "42"
	s.TokenHash = "abc"
	require.NoError(t, r.Save(t.Context(), s))

	got, err := r.Get(t.Context(), "42")
	require.NoError(t, err)
	require.Equal(t, s, got)
}

func TestGetShouldReturnErrorWhenContextIsCancelledForSession(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Get(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnErrorWhenContextIsCancelledForSession(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Delete(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnNotFoundWhenSessionDoesNotExist(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())

	err := r.Delete(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteShouldRemoveSessionWhenFound(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	var s domain.Session
	s.ID = "42"
	require.NoError(t, r.Save(t.Context(), s))

	require.NoError(t, r.Delete(t.Context(), "42"))

	_, err := r.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestFindByTokenHashShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.FindByTokenHash(ctx, "hash")
	require.ErrorIs(t, err, context.Canceled)
}

func TestFindByTokenHashShouldReturnNotFoundWhenHashDoesNotMatch(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())

	_, err := r.FindByTokenHash(t.Context(), "nonexistent")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestFindByTokenHashShouldReturnSessionWhenHashMatches(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	var s domain.Session
	s.ID = "42"
	s.TokenHash = "abc123"
	require.NoError(t, r.Save(t.Context(), s))

	got, err := r.FindByTokenHash(t.Context(), "abc123")
	require.NoError(t, err)
	require.Equal(t, s, got)
}

func TestCountByUserShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.CountByUser(ctx, "user1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestCountByUserShouldReturnZeroWhenUserHasNoSessions(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())

	count, err := r.CountByUser(t.Context(), "user1")
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestCountByUserShouldReturnCountOfUserSessions(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	var s1, s2, s3 domain.Session
	s1.ID = "1"
	s1.UserID = "user1"
	s2.ID = "2"
	s2.UserID = "user1"
	s3.ID = "3"
	s3.UserID = "user2"
	require.NoError(t, r.Save(t.Context(), s1))
	require.NoError(t, r.Save(t.Context(), s2))
	require.NoError(t, r.Save(t.Context(), s3))

	count, err := r.CountByUser(t.Context(), "user1")
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestDeleteOldestByUserShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.DeleteOldestByUser(ctx, "user1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteOldestByUserShouldReturnNotFoundWhenUserHasNoSessions(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())

	err := r.DeleteOldestByUser(t.Context(), "user1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteOldestByUserShouldDeleteOldestSession(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	now := time.Now()
	var oldest, newer domain.Session
	oldest.ID = "1"
	oldest.UserID = "user1"
	oldest.CreatedAt = now.Add(-time.Hour)
	newer.ID = "2"
	newer.UserID = "user1"
	newer.CreatedAt = now
	require.NoError(t, r.Save(t.Context(), oldest))
	require.NoError(t, r.Save(t.Context(), newer))

	require.NoError(t, r.DeleteOldestByUser(t.Context(), "user1"))

	_, err := r.Get(t.Context(), "1")
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = r.Get(t.Context(), "2")
	require.NoError(t, err)
}

func TestDeleteOldestByUserShouldDeleteOldestWhenNotInsertedFirst(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	now := time.Now()
	var recent, oldest, middle domain.Session
	recent.ID = "1"
	recent.UserID = "user1"
	recent.CreatedAt = now.Add(-30 * time.Minute)
	oldest.ID = "2"
	oldest.UserID = "user1"
	oldest.CreatedAt = now.Add(-2 * time.Hour)
	middle.ID = "3"
	middle.UserID = "user1"
	middle.CreatedAt = now.Add(-time.Hour)
	require.NoError(t, r.Save(t.Context(), recent))
	require.NoError(t, r.Save(t.Context(), oldest))
	require.NoError(t, r.Save(t.Context(), middle))

	require.NoError(t, r.DeleteOldestByUser(t.Context(), "user1"))

	_, err := r.Get(t.Context(), "2")
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = r.Get(t.Context(), "1")
	require.NoError(t, err)
	_, err = r.Get(t.Context(), "3")
	require.NoError(t, err)
}

func TestSaveShouldReturnErrorWhenContextIsCancelledForSession(t *testing.T) {
	r := json.NewSessionRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.Session{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestFindByTokenHashShouldReturnIOErrorWhenListFails(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sessions.json"), []byte("not json"), 0o644))
	r := json.NewSessionRepository(dir)

	_, err := r.FindByTokenHash(t.Context(), "hash")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestCountByUserShouldReturnIOErrorWhenListFails(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sessions.json"), []byte("not json"), 0o644))
	r := json.NewSessionRepository(dir)

	_, err := r.CountByUser(t.Context(), "user1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDeleteOldestByUserShouldReturnIOErrorWhenListFails(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sessions.json"), []byte("not json"), 0o644))
	r := json.NewSessionRepository(dir)

	err := r.DeleteOldestByUser(t.Context(), "user1")
	require.ErrorIs(t, err, domain.ErrIO)
}
