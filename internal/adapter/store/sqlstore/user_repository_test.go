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

func newUserRepo(t *testing.T) (*sqlstore.UserRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewUserRepository(db), db
}

func seedUser(t *testing.T, db *sql.DB, id, email string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (?, ?, 'hash', 'member', '2025-01-01T00:00:00Z')`,
		id, email)
	require.NoError(t, err)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestUserRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newUserRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "user-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserRepositoryGetShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newUserRepo(t)

	_, err := repo.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepositoryGetShouldReturnUserWhenRowExists(t *testing.T) {
	repo, db := newUserRepo(t)
	seedUser(t, db, "user-get-1", "get@example.com")

	got, err := repo.Get(t.Context(), "user-get-1")
	require.NoError(t, err)
	require.Equal(t, "user-get-1", got.GetID())
	require.Equal(t, "get@example.com", got.Email)
	require.Equal(t, "hash", got.PasswordHash)
	require.Equal(t, domain.RoleMember, got.Role)
	require.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), got.CreatedAt)
}

func TestUserRepositoryGetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewUserRepository(db)
	db.Close()

	_, err := repo.Get(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestUserRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newUserRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var u domain.User
	u.ID = "user-1"
	err := repo.Save(ctx, u)
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserRepositorySaveShouldInsertNewUser(t *testing.T) {
	repo, _ := newUserRepo(t)

	u := domain.User{}
	u.ID = "user-save-1"
	u.Email = "save@example.com"
	u.PasswordHash = "hashed"
	u.Role = domain.RoleAdmin
	u.CreatedAt = time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Save(t.Context(), u))

	got, err := repo.Get(t.Context(), "user-save-1")
	require.NoError(t, err)
	require.Equal(t, "save@example.com", got.Email)
	require.Equal(t, "hashed", got.PasswordHash)
	require.Equal(t, domain.RoleAdmin, got.Role)
}

func TestUserRepositorySaveShouldUpdateExistingUser(t *testing.T) {
	repo, db := newUserRepo(t)
	seedUser(t, db, "user-update-1", "old@example.com")

	var u domain.User
	u.ID = "user-update-1"
	u.Email = "new@example.com"
	u.PasswordHash = "newhash"
	u.Role = domain.RoleAdmin
	u.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Save(t.Context(), u))

	got, err := repo.Get(t.Context(), "user-update-1")
	require.NoError(t, err)
	require.Equal(t, "new@example.com", got.Email)
	require.Equal(t, "newhash", got.PasswordHash)
	require.Equal(t, domain.RoleAdmin, got.Role)
}

func TestUserRepositorySaveShouldReturnErrConflictWhenEmailTakenByDifferentUser(t *testing.T) {
	repo, db := newUserRepo(t)
	seedUser(t, db, "user-a", "taken@example.com")

	var u domain.User
	u.ID = "user-b"
	u.Email = "taken@example.com"
	u.PasswordHash = "hash"
	u.Role = domain.RoleMember
	u.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	err := repo.Save(t.Context(), u)
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestUserRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewUserRepository(db)
	db.Close()

	var u domain.User
	u.ID = "user-1"
	err := repo.Save(t.Context(), u)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── GetByEmail ────────────────────────────────────────────────────────────────

func TestUserRepositoryGetByEmailShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newUserRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.GetByEmail(ctx, "test@example.com")
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserRepositoryGetByEmailShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newUserRepo(t)

	_, err := repo.GetByEmail(t.Context(), "nobody@example.com")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepositoryGetByEmailShouldReturnUserWhenRowExists(t *testing.T) {
	repo, db := newUserRepo(t)
	seedUser(t, db, "user-email-1", "find@example.com")

	got, err := repo.GetByEmail(t.Context(), "find@example.com")
	require.NoError(t, err)
	require.Equal(t, "user-email-1", got.GetID())
	require.Equal(t, "find@example.com", got.Email)
}

func TestUserRepositoryGetByEmailShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewUserRepository(db)
	db.Close()

	_, err := repo.GetByEmail(t.Context(), "test@example.com")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── Count ─────────────────────────────────────────────────────────────────────

func TestUserRepositoryCountShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newUserRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Count(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserRepositoryCountShouldReturnZeroWhenNoRows(t *testing.T) {
	repo, _ := newUserRepo(t)

	count, err := repo.Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestUserRepositoryCountShouldReturnCorrectCount(t *testing.T) {
	repo, db := newUserRepo(t)
	seedUser(t, db, "user-c1", "c1@example.com")
	seedUser(t, db, "user-c2", "c2@example.com")

	count, err := repo.Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestUserRepositoryCountShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewUserRepository(db)
	db.Close()

	_, err := repo.Count(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestUserRepositoryListShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newUserRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestUserRepositoryListShouldReturnEmptySliceWhenNoRows(t *testing.T) {
	repo, _ := newUserRepo(t)

	users, err := repo.List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, users)
	require.Empty(t, users)
}

func TestUserRepositoryListShouldReturnAllUsers(t *testing.T) {
	repo, db := newUserRepo(t)
	seedUser(t, db, "user-l1", "l1@example.com")
	seedUser(t, db, "user-l2", "l2@example.com")

	users, err := repo.List(t.Context())
	require.NoError(t, err)
	require.Len(t, users, 2)

	ids := []string{users[0].GetID(), users[1].GetID()}
	require.ElementsMatch(t, []string{"user-l1", "user-l2"}, ids)
}

func TestUserRepositoryListShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewUserRepository(db)
	db.Close()

	_, err := repo.List(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUserRepositoryListShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	repo, db := newUserRepo(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-bad-ts', 'bad@example.com', 'hash', 'member', 'not-a-date')`)
	require.NoError(t, err)

	_, err = repo.List(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}
