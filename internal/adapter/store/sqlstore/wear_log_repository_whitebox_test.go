package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/domain"
)

// ── Delete: RowsAffected error ─────────────────────────────────────────────────

func TestWearLogRepositoryDeleteShouldReturnErrIOWhenRowsAffectedFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-aff-err")
	repo := &WearLogRepository{db: db}
	err := repo.Delete(t.Context(), "log-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanWearLogRow: scan error (non-ErrNoRows) ────────────────────────────────

func TestScanWearLogRowShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)
	// SELECT 1 returns a single integer column; scanning into 6 fields will fail.
	row := db.QueryRowContext(t.Context(), "SELECT 1")
	_, err := scanWearLogRow(row)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanWearLogRow: worn_on time.Parse error ───────────────────────────────────

func TestScanWearLogRowShouldReturnErrIOWhenWornOnIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('owner-bad-worn', 'owner-bad-worn@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-bad-worn', 'owner-bad-worn', 'Item', '2025-01-01T00:00:00Z', '{}')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES ('log-bad-worn', 'item-bad-worn', 'owner-bad-worn', 'not-a-date', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	repo := &WearLogRepository{db: db}
	_, err = repo.Get(t.Context(), "log-bad-worn")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanWearLogRow: created_at time.Parse error ────────────────────────────────

func TestScanWearLogRowShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('owner-bad-ca', 'owner-bad-ca@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-bad-ca', 'owner-bad-ca', 'Item', '2025-01-01T00:00:00Z', '{}')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES ('log-bad-ca', 'item-bad-ca', 'owner-bad-ca', '2025-01-01', 'not-a-timestamp')`)
	require.NoError(t, err)

	repo := &WearLogRepository{db: db}
	_, err = repo.Get(t.Context(), "log-bad-ca")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanWearLogRows: rows.Err error ───────────────────────────────────────────

func TestWearLogRepositoryListByItemShouldReturnErrIOWhenRowsErrFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &WearLogRepository{db: db}
	_, err := repo.ListByItem(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanWearLogRows: scan error ───────────────────────────────────────────────

func TestWearLogRepositoryListByItemShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &WearLogRepository{db: db}
	_, err := repo.ListByItem(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanWearLogRows: worn_on time.Parse error ─────────────────────────────────

func TestWearLogRepositoryListByItemShouldReturnErrIOWhenWornOnIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('owner-list-bad-worn', 'owner-list-bad-worn@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-list-bad-worn', 'owner-list-bad-worn', 'Item', '2025-01-01T00:00:00Z', '{}')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES ('log-list-bad-worn', 'item-list-bad-worn', 'owner-list-bad-worn', 'bad-date', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	repo := &WearLogRepository{db: db}
	_, err = repo.ListByItem(t.Context(), "item-list-bad-worn")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanWearLogRows: created_at time.Parse error ──────────────────────────────

func TestWearLogRepositoryListByItemShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('owner-list-bad-ca', 'owner-list-bad-ca@example.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-list-bad-ca', 'owner-list-bad-ca', 'Item', '2025-01-01T00:00:00Z', '{}')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at)
		VALUES ('log-list-bad-ca', 'item-list-bad-ca', 'owner-list-bad-ca', '2025-01-01', 'bad-timestamp')`)
	require.NoError(t, err)

	repo := &WearLogRepository{db: db}
	_, err = repo.ListByItem(t.Context(), "item-list-bad-ca")
	require.ErrorIs(t, err, domain.ErrIO)
}
