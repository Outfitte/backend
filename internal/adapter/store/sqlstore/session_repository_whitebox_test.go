package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/domain"
)

// ── Delete rows.RowsAffected error ────────────────────────────────────────────

func TestSessionRepositoryDeleteShouldReturnErrIOWhenRowsAffectedFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-aff-err")
	repo := &SessionRepository{db: db}
	err := repo.Delete(t.Context(), "session-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── DeleteOldestByUser rows.RowsAffected error ────────────────────────────────

func TestSessionRepositoryDeleteOldestByUserShouldReturnErrIOWhenRowsAffectedFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-aff-err")
	repo := &SessionRepository{db: db}
	err := repo.DeleteOldestByUser(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanSessionRow timestamp parse errors ─────────────────────────────────────

func TestScanSessionRowShouldReturnErrIOWhenExpiresAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-ts-1', 'ts@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES ('session-bad-exp', 'user-ts-1', 'hash-bad-exp', 'not-a-date', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	row := db.QueryRowContext(t.Context(),
		`SELECT id, user_id, token_hash, expires_at, created_at FROM sessions WHERE id = ?`,
		"session-bad-exp")
	_, err = scanSessionRow(row)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestScanSessionRowShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-ts-2', 'ts2@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES ('session-bad-cat', 'user-ts-2', 'hash-bad-cat', '2030-01-01T00:00:00Z', 'not-a-date')`)
	require.NoError(t, err)

	row := db.QueryRowContext(t.Context(),
		`SELECT id, user_id, token_hash, expires_at, created_at FROM sessions WHERE id = ?`,
		"session-bad-cat")
	_, err = scanSessionRow(row)
	require.ErrorIs(t, err, domain.ErrIO)
}
