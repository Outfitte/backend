package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/domain"
)

// ── List rows.Err ─────────────────────────────────────────────────────────────

func TestUserRepositoryListRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &UserRepository{db: db}
	_, err := repo.List(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── List rows.Scan error ──────────────────────────────────────────────────────

func TestUserRepositoryListShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &UserRepository{db: db}
	_, err := repo.List(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanUserRow created_at parse error ───────────────────────────────────────

func TestScanUserRowShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-bad-ts', 'bad@example.com', 'hash', 'member', 'not-a-date')`)
	require.NoError(t, err)

	row := db.QueryRowContext(t.Context(),
		`SELECT id, email, password_hash, role, created_at FROM users WHERE id = ?`,
		"user-bad-ts")
	_, err = scanUserRow(row)
	require.ErrorIs(t, err, domain.ErrIO)
}
