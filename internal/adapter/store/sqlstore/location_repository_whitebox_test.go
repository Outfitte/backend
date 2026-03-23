package sqlstore

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/domain"
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
		INSERT INTO locations (id, owner_id, label, created_at)
		VALUES ('loc-bad-ts', 'owner-1', 'Bad', 'not-a-date')`)
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
		INSERT INTO locations (id, owner_id, label, created_at)
		VALUES ('loc-bad-list-ts', 'owner-bad', 'Bad', 'not-a-date')`)
	require.NoError(t, err)

	repo := &LocationRepository{db: db}
	_, err = repo.ListByOwner(t.Context(), "owner-bad")
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
	// SELECT 1 returns a single integer column; scanning into (string, string, NullString, string, string) will fail.
	rows, err := db.QueryContext(t.Context(), "SELECT 1")
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())
	row := &sql.Row{}
	_ = row // scanLocationRow takes a *sql.Row, use QueryRowContext with wrong columns instead
	rows.Close()

	// Query a location with too few columns to cause a scan error.
	row2 := db.QueryRowContext(t.Context(), "SELECT 1")
	_, err = scanLocationRow(row2)
	require.ErrorIs(t, err, domain.ErrIO)
}
