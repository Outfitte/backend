package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/domain"
)

// ── Delete: RowsAffected error ─────────────────────────────────────────────────

func TestLocationRepositoryDeleteShouldReturnErrIOWhenRowsAffectedFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-aff-err")
	repo := &LocationRepository{db: db}
	err := repo.Delete(t.Context(), "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanLocationRow: time.Parse error ─────────────────────────────────────────

func TestScanLocationRowShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('owner-bad-ts', 'owner-bad-ts@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO locations (id, owner_id, label, created_at)
		VALUES ('loc-bad-ts', 'owner-bad-ts', 'Bad', 'not-a-date')`)
	require.NoError(t, err)

	repo := &LocationRepository{db: db}
	_, err = repo.Get(t.Context(), "loc-bad-ts")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanLocationRows: rows.Err and time.Parse errors ──────────────────────────

func TestLocationRepositoryListByOwnerShouldReturnErrIOWhenRowsErrFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &LocationRepository{db: db}
	_, err := repo.ListByOwner(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationRepositoryListByOwnerShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('owner-bad-list', 'owner-bad-list@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO locations (id, owner_id, label, created_at)
		VALUES ('loc-bad-list-ts', 'owner-bad-list', 'Bad', 'not-a-date')`)
	require.NoError(t, err)

	repo := &LocationRepository{db: db}
	_, err = repo.ListByOwner(t.Context(), "owner-bad-list")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanLocationRows: scan error ──────────────────────────────────────────────

func TestLocationRepositoryListByOwnerShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &LocationRepository{db: db}
	_, err := repo.ListByOwner(t.Context(), "user-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanLocationRow: scan error (non-ErrNoRows) ───────────────────────────────

func TestScanLocationRowShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)
	// SELECT 1 returns a single integer column; scanning into 5 fields will fail.
	row := db.QueryRowContext(t.Context(), "SELECT 1")
	_, err := scanLocationRow(row)
	require.ErrorIs(t, err, domain.ErrIO)
}
