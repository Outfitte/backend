package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/domain"
)

// ── Delete: RowsAffected error ─────────────────────────────────────────────────

func TestShareRepositoryDeleteShouldReturnErrIOWhenRowsAffectedFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-aff-err")
	repo := &ShareRepository{db: db}
	err := repo.Delete(t.Context(), "share-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanShareRow: time.Parse error ────────────────────────────────────────────

func TestScanShareRowShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('owner-bad-ts-sh', 'owner-bad-ts-sh@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('recip-bad-ts-sh', 'recip-bad-ts-sh@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO shares (id, owner_id, recipient_id, target_type, target_id, created_at)
		VALUES ('share-bad-ts', 'owner-bad-ts-sh', 'recip-bad-ts-sh', 'item', 'item-1', 'not-a-date')`)
	require.NoError(t, err)

	repo := &ShareRepository{db: db}
	_, err = repo.Get(t.Context(), "share-bad-ts")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanShareRows: rows.Err and time.Parse errors ─────────────────────────────

func TestShareRepositoryListByOwnerShouldReturnErrIOWhenRowsErrFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &ShareRepository{db: db}
	_, err := repo.ListByOwner(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestShareRepositoryListByOwnerShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('owner-bad-list-sh', 'owner-bad-list-sh@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('recip-bad-list-sh', 'recip-bad-list-sh@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO shares (id, owner_id, recipient_id, target_type, target_id, created_at)
		VALUES ('share-bad-list-ts', 'owner-bad-list-sh', 'recip-bad-list-sh', 'item', 'item-1', 'not-a-date')`)
	require.NoError(t, err)

	repo := &ShareRepository{db: db}
	_, err = repo.ListByOwner(t.Context(), "owner-bad-list-sh")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanShareRows: scan error ─────────────────────────────────────────────────

func TestShareRepositoryListByOwnerShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &ShareRepository{db: db}
	_, err := repo.ListByOwner(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanShareRow: scan error (non-ErrNoRows) ──────────────────────────────────

func TestScanShareRowShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)
	// SELECT 1 returns a single integer column; scanning into 6 fields will fail.
	row := db.QueryRowContext(t.Context(), "SELECT 1")
	_, err := scanShareRow(row)
	require.ErrorIs(t, err, domain.ErrIO)
}
