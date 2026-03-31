package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/adapter/store/sqlstore"
	"github.com/outfitte/backend/internal/domain"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newSessionRepo(t *testing.T) (*sqlstore.SessionRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewSessionRepository(db), db
}

func seedSession(t *testing.T, db *sql.DB, id, userID, tokenHash string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, '2030-01-01T00:00:00Z', '2025-01-01T00:00:00Z')`,
		id, userID, tokenHash)
	require.NoError(t, err)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestSessionRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newSessionRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "session-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSessionRepositoryGetShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newSessionRepo(t)

	_, err := repo.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSessionRepositoryGetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewSessionRepository(db)
	db.Close()

	_, err := repo.Get(t.Context(), "session-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestSessionRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newSessionRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var s domain.Session
	s.ID = "session-1"
	err := repo.Save(ctx, s)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSessionRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewSessionRepository(db)
	db.Close()

	var s domain.Session
	s.ID = "session-1"
	err := repo.Save(t.Context(), s)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSessionRepositorySaveShouldInsertNewSession(t *testing.T) {
	repo, db := newSessionRepo(t)
	seedUser(t, db, "user-save-1", "save@example.com")

	var s domain.Session
	s.ID = "session-save-1"
	s.UserID = "user-save-1"
	s.TokenHash = "hash-save-1"
	s.ExpiresAt = time.Date(2030, 3, 1, 0, 0, 0, 0, time.UTC)
	s.CreatedAt = time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Save(t.Context(), s))

	got, err := repo.Get(t.Context(), "session-save-1")
	require.NoError(t, err)
	require.Equal(t, "session-save-1", got.GetID())
	require.Equal(t, "user-save-1", got.UserID)
	require.Equal(t, "hash-save-1", got.TokenHash)
	require.Equal(t, time.Date(2030, 3, 1, 0, 0, 0, 0, time.UTC), got.ExpiresAt)
}

func TestSessionRepositorySaveShouldUpdateExistingSession(t *testing.T) {
	repo, db := newSessionRepo(t)
	seedUser(t, db, "user-upd-1", "upd@example.com")
	// seedSession inserts created_at = 2025-01-01; the upsert must not change it.
	seedSession(t, db, "session-upd-1", "user-upd-1", "old-hash")

	var s domain.Session
	s.ID = "session-upd-1"
	s.UserID = "user-upd-1"
	s.TokenHash = "new-hash"
	s.ExpiresAt = time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)
	s.CreatedAt = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // different from seeded value

	require.NoError(t, repo.Save(t.Context(), s))

	got, err := repo.Get(t.Context(), "session-upd-1")
	require.NoError(t, err)
	require.Equal(t, "new-hash", got.TokenHash)
	require.Equal(t, time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC), got.ExpiresAt)
	require.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), got.CreatedAt) // must not change
}

// ── DeleteOldestByUser ────────────────────────────────────────────────────────

func TestSessionRepositoryDeleteOldestByUserShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newSessionRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.DeleteOldestByUser(ctx, "user-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSessionRepositoryDeleteOldestByUserShouldReturnErrNotFoundWhenUserHasNoSessions(t *testing.T) {
	repo, _ := newSessionRepo(t)

	err := repo.DeleteOldestByUser(t.Context(), "nonexistent-user")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSessionRepositoryDeleteOldestByUserShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewSessionRepository(db)
	db.Close()

	err := repo.DeleteOldestByUser(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSessionRepositoryDeleteOldestByUserShouldRemoveOldestSession(t *testing.T) {
	repo, db := newSessionRepo(t)
	seedUser(t, db, "user-dob-1", "dob@example.com")

	// Insert two sessions with different created_at timestamps.
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES ('session-dob-old', 'user-dob-1', 'hash-dob-old', '2030-01-01T00:00:00Z', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES ('session-dob-new', 'user-dob-1', 'hash-dob-new', '2030-01-01T00:00:00Z', '2025-06-01T00:00:00Z')`)
	require.NoError(t, err)

	require.NoError(t, repo.DeleteOldestByUser(t.Context(), "user-dob-1"))

	// Oldest session should be gone.
	_, err = repo.Get(t.Context(), "session-dob-old")
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Newer session should still exist.
	_, err = repo.Get(t.Context(), "session-dob-new")
	require.NoError(t, err)
}

// ── CountByUser ───────────────────────────────────────────────────────────────

func TestSessionRepositoryCountByUserShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newSessionRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.CountByUser(ctx, "user-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSessionRepositoryCountByUserShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewSessionRepository(db)
	db.Close()

	_, err := repo.CountByUser(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSessionRepositoryCountByUserShouldReturnZeroWhenUserHasNoSessions(t *testing.T) {
	repo, _ := newSessionRepo(t)

	count, err := repo.CountByUser(t.Context(), "nonexistent-user")
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestSessionRepositoryCountByUserShouldReturnCorrectCount(t *testing.T) {
	repo, db := newSessionRepo(t)
	seedUser(t, db, "user-cnt-1", "cnt@example.com")
	seedSession(t, db, "session-cnt-1", "user-cnt-1", "hash-cnt-1")
	seedSession(t, db, "session-cnt-2", "user-cnt-1", "hash-cnt-2")

	count, err := repo.CountByUser(t.Context(), "user-cnt-1")
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

// ── FindByTokenHash ───────────────────────────────────────────────────────────

func TestSessionRepositoryFindByTokenHashShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newSessionRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.FindByTokenHash(ctx, "some-hash")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSessionRepositoryFindByTokenHashShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newSessionRepo(t)

	_, err := repo.FindByTokenHash(t.Context(), "nonexistent-hash")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSessionRepositoryFindByTokenHashShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewSessionRepository(db)
	db.Close()

	_, err := repo.FindByTokenHash(t.Context(), "some-hash")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSessionRepositoryFindByTokenHashShouldReturnSessionWhenHashExists(t *testing.T) {
	repo, db := newSessionRepo(t)
	seedUser(t, db, "user-fth-1", "fth@example.com")
	seedSession(t, db, "session-fth-1", "user-fth-1", "token-hash-fth")

	got, err := repo.FindByTokenHash(t.Context(), "token-hash-fth")
	require.NoError(t, err)
	require.Equal(t, "session-fth-1", got.GetID())
	require.Equal(t, "user-fth-1", got.UserID)
	require.Equal(t, "token-hash-fth", got.TokenHash)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestSessionRepositoryDeleteShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newSessionRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.Delete(ctx, "session-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSessionRepositoryDeleteShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newSessionRepo(t)

	err := repo.Delete(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSessionRepositoryDeleteShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewSessionRepository(db)
	db.Close()

	err := repo.Delete(t.Context(), "session-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSessionRepositoryDeleteShouldRemoveSessionWhenRowExists(t *testing.T) {
	repo, db := newSessionRepo(t)
	seedUser(t, db, "user-del-1", "del@example.com")
	seedSession(t, db, "session-del-1", "user-del-1", "hash-del-1")

	require.NoError(t, repo.Delete(t.Context(), "session-del-1"))

	_, err := repo.Get(t.Context(), "session-del-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSessionRepositoryGetShouldReturnSessionWhenRowExists(t *testing.T) {
	repo, db := newSessionRepo(t)
	seedUser(t, db, "user-get-1", "get@example.com")
	seedSession(t, db, "session-get-1", "user-get-1", "hash-get-1")

	got, err := repo.Get(t.Context(), "session-get-1")
	require.NoError(t, err)
	require.Equal(t, "session-get-1", got.GetID())
	require.Equal(t, "user-get-1", got.UserID)
	require.Equal(t, "hash-get-1", got.TokenHash)
	require.Equal(t, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), got.ExpiresAt)
	require.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), got.CreatedAt)
}
